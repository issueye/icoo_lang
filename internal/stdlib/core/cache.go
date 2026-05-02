package core

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"
)

type cacheBinding struct {
	mu         sync.Mutex
	defaultTTL time.Duration
	maxEntries int
	touchOnGet bool
	sequence   int64
	hits       int64
	misses     int64
	evictions  int64
	items      map[string]cacheEntry
}

type cacheEntry struct {
	value     runtime.Value
	expiresAt time.Time
	order     int64
}

func LoadStdCacheModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.cache",
		Path: "std.cache",
		Exports: map[string]runtime.Value{
			"create": &runtime.NativeFunction{Name: "create", Arity: -1, Fn: cacheCreate},
		},
		Done: true,
	}
}

func cacheCreate(args []runtime.Value) (runtime.Value, error) {
	binding := &cacheBinding{
		items: map[string]cacheEntry{},
	}
	if len(args) > 1 {
		return nil, fmt.Errorf("create expects optional options argument")
	}
	if len(args) == 1 {
		switch value := args[0].(type) {
		case runtime.IntValue:
			if value.Value < 0 {
				return nil, fmt.Errorf("create maxEntries must be non-negative")
			}
			binding.maxEntries = int(value.Value)
		case *runtime.ObjectValue:
			if err := binding.applyOptions(value); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("create expects int or options object")
		}
	}
	return binding.object(), nil
}

func (binding *cacheBinding) applyOptions(options *runtime.ObjectValue) error {
	if ttlValue, ok := options.Fields["defaultTTL"]; ok {
		ms, err := requireCacheIntArg("create", ttlValue)
		if err != nil {
			return err
		}
		if ms < 0 {
			return fmt.Errorf("create defaultTTL must be non-negative")
		}
		binding.defaultTTL = time.Duration(ms) * time.Millisecond
	}
	if maxEntriesValue, ok := options.Fields["maxEntries"]; ok {
		maxEntries, err := requireCacheIntArg("create", maxEntriesValue)
		if err != nil {
			return err
		}
		if maxEntries < 0 {
			return fmt.Errorf("create maxEntries must be non-negative")
		}
		binding.maxEntries = int(maxEntries)
	}
	if touchValue, ok := options.Fields["touchOnGet"]; ok {
		boolValue, ok := touchValue.(runtime.BoolValue)
		if !ok {
			return fmt.Errorf("create touchOnGet must be bool")
		}
		binding.touchOnGet = boolValue.Value
	}
	return nil
}

func (binding *cacheBinding) object() *runtime.ObjectValue {
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"clear": &runtime.NativeFunction{Name: "cache.clear", Arity: 0, Fn: binding.clear},
		"del":   &runtime.NativeFunction{Name: "cache.del", Arity: 1, Fn: binding.del},
		"get":   &runtime.NativeFunction{Name: "cache.get", Arity: -1, Fn: binding.get},
		"has":   &runtime.NativeFunction{Name: "cache.has", Arity: 1, Fn: binding.has},
		"keys":  &runtime.NativeFunction{Name: "cache.keys", Arity: 0, Fn: binding.keys},
		"set":   &runtime.NativeFunction{Name: "cache.set", Arity: -1, Fn: binding.set},
		"size":  &runtime.NativeFunction{Name: "cache.size", Arity: 0, Fn: binding.size},
		"stats": &runtime.NativeFunction{Name: "cache.stats", Arity: 0, Fn: binding.stats},
	}}
}

func (binding *cacheBinding) set(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("set expects key, value, and optional ttl")
	}
	key, err := utils.RequireStringArg("set", args[0])
	if err != nil {
		return nil, err
	}
	value, err := cloneCacheValue(args[1])
	if err != nil {
		return nil, err
	}
	ttl := binding.defaultTTL
	if len(args) == 3 {
		ms, err := requireCacheIntArg("set", args[2])
		if err != nil {
			return nil, err
		}
		if ms < 0 {
			return nil, fmt.Errorf("set ttl must be non-negative")
		}
		ttl = time.Duration(ms) * time.Millisecond
	}

	binding.mu.Lock()
	defer binding.mu.Unlock()
	binding.purgeExpiredLocked()
	binding.sequence++
	entry := cacheEntry{value: value, order: binding.sequence}
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}
	binding.items[key] = entry
	binding.enforceLimitLocked()
	return binding.object(), nil
}

func (binding *cacheBinding) get(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("get expects key and optional fallback")
	}
	key, err := utils.RequireStringArg("get", args[0])
	if err != nil {
		return nil, err
	}
	fallback := runtime.Value(runtime.NullValue{})
	if len(args) == 2 {
		fallback = args[1]
	}

	binding.mu.Lock()
	defer binding.mu.Unlock()
	binding.purgeExpiredLocked()
	entry, ok := binding.items[key]
	if !ok {
		binding.misses++
		return fallback, nil
	}
	if binding.touchOnGet && !entry.expiresAt.IsZero() && binding.defaultTTL > 0 {
		entry.expiresAt = time.Now().Add(binding.defaultTTL)
		binding.items[key] = entry
	}
	binding.hits++
	return entry.value, nil
}

func (binding *cacheBinding) has(args []runtime.Value) (runtime.Value, error) {
	key, err := utils.RequireStringArg("has", args[0])
	if err != nil {
		return nil, err
	}
	binding.mu.Lock()
	defer binding.mu.Unlock()
	binding.purgeExpiredLocked()
	_, ok := binding.items[key]
	return runtime.BoolValue{Value: ok}, nil
}

func (binding *cacheBinding) del(args []runtime.Value) (runtime.Value, error) {
	key, err := utils.RequireStringArg("del", args[0])
	if err != nil {
		return nil, err
	}
	binding.mu.Lock()
	defer binding.mu.Unlock()
	binding.purgeExpiredLocked()
	_, ok := binding.items[key]
	delete(binding.items, key)
	return runtime.BoolValue{Value: ok}, nil
}

func (binding *cacheBinding) clear(args []runtime.Value) (runtime.Value, error) {
	binding.mu.Lock()
	defer binding.mu.Unlock()
	binding.items = map[string]cacheEntry{}
	return runtime.NullValue{}, nil
}

func (binding *cacheBinding) size(args []runtime.Value) (runtime.Value, error) {
	binding.mu.Lock()
	defer binding.mu.Unlock()
	binding.purgeExpiredLocked()
	return runtime.IntValue{Value: int64(len(binding.items))}, nil
}

func (binding *cacheBinding) keys(args []runtime.Value) (runtime.Value, error) {
	binding.mu.Lock()
	defer binding.mu.Unlock()
	binding.purgeExpiredLocked()
	keys := make([]string, 0, len(binding.items))
	for key := range binding.items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]runtime.Value, 0, len(keys))
	for _, key := range keys {
		values = append(values, runtime.StringValue{Value: key})
	}
	return &runtime.ArrayValue{Elements: values}, nil
}

func (binding *cacheBinding) stats(args []runtime.Value) (runtime.Value, error) {
	binding.mu.Lock()
	defer binding.mu.Unlock()
	binding.purgeExpiredLocked()
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"size":       runtime.IntValue{Value: int64(len(binding.items))},
		"hits":       runtime.IntValue{Value: binding.hits},
		"misses":     runtime.IntValue{Value: binding.misses},
		"evictions":  runtime.IntValue{Value: binding.evictions},
		"maxEntries": runtime.IntValue{Value: int64(binding.maxEntries)},
		"defaultTTL": runtime.IntValue{Value: binding.defaultTTL.Milliseconds()},
	}}, nil
}

func (binding *cacheBinding) purgeExpiredLocked() {
	if len(binding.items) == 0 {
		return
	}
	now := time.Now()
	for key, entry := range binding.items {
		if !entry.expiresAt.IsZero() && !entry.expiresAt.After(now) {
			delete(binding.items, key)
		}
	}
}

func (binding *cacheBinding) enforceLimitLocked() {
	if binding.maxEntries <= 0 || len(binding.items) <= binding.maxEntries {
		return
	}
	var oldestKey string
	var oldestOrder int64
	first := true
	for key, entry := range binding.items {
		if first || entry.order < oldestOrder {
			first = false
			oldestKey = key
			oldestOrder = entry.order
		}
	}
	if oldestKey != "" {
		delete(binding.items, oldestKey)
		binding.evictions++
	}
}

func requireCacheIntArg(name string, value runtime.Value) (int64, error) {
	intValue, ok := value.(runtime.IntValue)
	if !ok {
		return 0, fmt.Errorf("%s expects int argument", name)
	}
	return intValue.Value, nil
}

func cloneCacheValue(value runtime.Value) (runtime.Value, error) {
	plain, err := utils.RuntimeToPlainValue(value)
	if err != nil {
		switch typed := value.(type) {
		case runtime.NullValue, runtime.BoolValue, runtime.IntValue, runtime.FloatValue, runtime.StringValue:
			return typed, nil
		default:
			return nil, err
		}
	}
	return utils.PlainToRuntimeValue(plain), nil
}
