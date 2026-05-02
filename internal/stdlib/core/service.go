package core

import (
	"fmt"
	"sync"
	"time"

	"icoo_lang/internal/runtime"
)

type serviceBinding struct {
	mu          sync.RWMutex
	name        string
	startedAt   int64
	ready       bool
	readyReason string
	recent      *observeRecentBinding
}

func LoadStdServiceModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.service",
		Path: "std.service",
		Exports: map[string]runtime.Value{
			"create": &runtime.NativeFunction{Name: "create", Arity: 1, Fn: serviceCreate},
		},
		Done: true,
	}
}

func serviceCreate(args []runtime.Value) (runtime.Value, error) {
	options, ok := args[0].(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("create expects options object")
	}

	nameValue, ok := options.Fields["name"].(runtime.StringValue)
	if !ok || nameValue.Value == "" {
		return nil, fmt.Errorf("create options require non-empty name")
	}

	recentLimit := int64(0)
	if recentLimitValue, ok := options.Fields["recentLimit"]; ok {
		intValue, ok := recentLimitValue.(runtime.IntValue)
		if !ok {
			return nil, fmt.Errorf("create recentLimit must be int")
		}
		if intValue.Value < 0 {
			return nil, fmt.Errorf("create recentLimit must be non-negative")
		}
		recentLimit = intValue.Value
	}

	ready := true
	if readyValue, ok := options.Fields["ready"]; ok {
		boolValue, ok := readyValue.(runtime.BoolValue)
		if !ok {
			return nil, fmt.Errorf("create ready must be bool")
		}
		ready = boolValue.Value
	}

	binding := &serviceBinding{
		name:      nameValue.Value,
		startedAt: time.Now().UnixMilli(),
		ready:     ready,
		recent: &observeRecentBinding{
			limit: int(recentLimit),
			items: []runtime.Value{},
		},
	}
	return binding.object(), nil
}

func (binding *serviceBinding) object() *runtime.ObjectValue {
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"name":           runtime.StringValue{Value: binding.name},
		"startedAt":      runtime.IntValue{Value: binding.startedAt},
		"health":         &runtime.NativeFunction{Name: "service.health", Arity: 0, Fn: binding.health},
		"ready":          &runtime.NativeFunction{Name: "service.ready", Arity: 0, Fn: binding.readyStatus},
		"markReady":      &runtime.NativeFunction{Name: "service.markReady", Arity: 0, Fn: binding.markReady},
		"markNotReady":   &runtime.NativeFunction{Name: "service.markNotReady", Arity: -1, Fn: binding.markNotReady},
		"recordRequest":  &runtime.NativeFunction{Name: "service.recordRequest", Arity: 1, Fn: binding.recordRequest},
		"clearRequests":  &runtime.NativeFunction{Name: "service.clearRequests", Arity: 0, Fn: binding.clearRequests},
		"requestCount":   &runtime.NativeFunction{Name: "service.requestCount", Arity: 0, Fn: binding.requestCount},
		"recentRequests": &runtime.NativeFunction{Name: "service.recentRequests", Arity: 0, Fn: binding.recentRequests},
		"snapshot":       &runtime.NativeFunction{Name: "service.snapshot", Arity: 0, Fn: binding.snapshot},
	}}
}

func (binding *serviceBinding) health(args []runtime.Value) (runtime.Value, error) {
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"ok":        runtime.BoolValue{Value: true},
		"service":   runtime.StringValue{Value: binding.name},
		"startedAt": runtime.IntValue{Value: binding.startedAt},
		"uptimeMs":  runtime.IntValue{Value: binding.uptimeMs()},
	}}, nil
}

func (binding *serviceBinding) readyStatus(args []runtime.Value) (runtime.Value, error) {
	binding.mu.RLock()
	ready := binding.ready
	reason := binding.readyReason
	binding.mu.RUnlock()

	reasonValue := runtime.Value(runtime.NullValue{})
	if reason != "" {
		reasonValue = runtime.StringValue{Value: reason}
	}
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"ok":        runtime.BoolValue{Value: ready},
		"service":   runtime.StringValue{Value: binding.name},
		"startedAt": runtime.IntValue{Value: binding.startedAt},
		"uptimeMs":  runtime.IntValue{Value: binding.uptimeMs()},
		"reason":    reasonValue,
	}}, nil
}

func (binding *serviceBinding) markReady(args []runtime.Value) (runtime.Value, error) {
	binding.mu.Lock()
	defer binding.mu.Unlock()
	binding.ready = true
	binding.readyReason = ""
	return runtime.NullValue{}, nil
}

func (binding *serviceBinding) markNotReady(args []runtime.Value) (runtime.Value, error) {
	if len(args) > 1 {
		return nil, fmt.Errorf("markNotReady expects 0 or 1 arguments")
	}
	reason := ""
	if len(args) == 1 {
		text, ok := args[0].(runtime.StringValue)
		if !ok {
			return nil, fmt.Errorf("markNotReady reason must be string")
		}
		reason = text.Value
	}
	binding.mu.Lock()
	defer binding.mu.Unlock()
	binding.ready = false
	binding.readyReason = reason
	return runtime.NullValue{}, nil
}

func (binding *serviceBinding) recordRequest(args []runtime.Value) (runtime.Value, error) {
	return binding.recent.add(args)
}

func (binding *serviceBinding) clearRequests(args []runtime.Value) (runtime.Value, error) {
	return binding.recent.clear(args)
}

func (binding *serviceBinding) requestCount(args []runtime.Value) (runtime.Value, error) {
	return binding.recent.totalCount(args)
}

func (binding *serviceBinding) recentRequests(args []runtime.Value) (runtime.Value, error) {
	return binding.recent.list(args)
}

func (binding *serviceBinding) snapshot(args []runtime.Value) (runtime.Value, error) {
	healthValue, err := binding.health(nil)
	if err != nil {
		return nil, err
	}
	readyValue, err := binding.readyStatus(nil)
	if err != nil {
		return nil, err
	}
	requestCount, err := binding.requestCount(nil)
	if err != nil {
		return nil, err
	}
	recentRequests, err := binding.recentRequests(nil)
	if err != nil {
		return nil, err
	}
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"service":       runtime.StringValue{Value: binding.name},
		"startedAt":     runtime.IntValue{Value: binding.startedAt},
		"uptimeMs":      runtime.IntValue{Value: binding.uptimeMs()},
		"health":        healthValue,
		"ready":         readyValue,
		"requestCount":  requestCount,
		"recentRequests": recentRequests,
	}}, nil
}

func (binding *serviceBinding) uptimeMs() int64 {
	now := time.Now().UnixMilli()
	if now < binding.startedAt {
		return 0
	}
	return now - binding.startedAt
}
