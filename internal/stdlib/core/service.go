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
	counters    map[string]int64
	latency     serviceLatencyStats
	recent      *observeRecentBinding
}

type serviceLatencyStats struct {
	count   int64
	totalMs int64
	maxMs   int64
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
		counters:  map[string]int64{},
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
		"increment":      &runtime.NativeFunction{Name: "service.increment", Arity: -1, Fn: binding.increment},
		"counter":        &runtime.NativeFunction{Name: "service.counter", Arity: 1, Fn: binding.counter},
		"counters":       &runtime.NativeFunction{Name: "service.counters", Arity: 0, Fn: binding.countersSnapshot},
		"recordRequest":  &runtime.NativeFunction{Name: "service.recordRequest", Arity: 1, Fn: binding.recordRequest},
		"clearRequests":  &runtime.NativeFunction{Name: "service.clearRequests", Arity: 0, Fn: binding.clearRequests},
		"requestCount":   &runtime.NativeFunction{Name: "service.requestCount", Arity: 0, Fn: binding.requestCount},
		"recentRequests": &runtime.NativeFunction{Name: "service.recentRequests", Arity: 0, Fn: binding.recentRequests},
		"latency":        &runtime.NativeFunction{Name: "service.latency", Arity: 0, Fn: binding.latencySnapshot},
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

func (binding *serviceBinding) increment(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("increment expects name and optional delta")
	}

	nameValue, ok := args[0].(runtime.StringValue)
	if !ok || nameValue.Value == "" {
		return nil, fmt.Errorf("increment name must be non-empty string")
	}

	delta := int64(1)
	if len(args) == 2 {
		intValue, ok := args[1].(runtime.IntValue)
		if !ok {
			return nil, fmt.Errorf("increment delta must be int")
		}
		if intValue.Value < 0 {
			return nil, fmt.Errorf("increment delta must be non-negative")
		}
		delta = intValue.Value
	}

	return runtime.IntValue{Value: binding.incrementCounter(nameValue.Value, delta)}, nil
}

func (binding *serviceBinding) counter(args []runtime.Value) (runtime.Value, error) {
	nameValue, ok := args[0].(runtime.StringValue)
	if !ok || nameValue.Value == "" {
		return nil, fmt.Errorf("counter expects non-empty string name")
	}
	return runtime.IntValue{Value: binding.counterValue(nameValue.Value)}, nil
}

func (binding *serviceBinding) countersSnapshot(args []runtime.Value) (runtime.Value, error) {
	return binding.countersObject(), nil
}

func (binding *serviceBinding) recordRequest(args []runtime.Value) (runtime.Value, error) {
	recordValue, statusCode, durationMs, err := binding.normalizeRequestRecord(args[0])
	if err != nil {
		return nil, err
	}
	if _, err := binding.recent.add([]runtime.Value{recordValue}); err != nil {
		return nil, err
	}

	binding.incrementCounter("requests.total", 1)
	if statusCode > 0 {
		binding.incrementCounter(fmt.Sprintf("requests.status.%d", statusCode), 1)
		binding.incrementCounter(fmt.Sprintf("requests.status.%dxx", statusCode/100), 1)
	}
	if durationMs >= 0 {
		binding.observeLatency(durationMs)
	}
	return runtime.NullValue{}, nil
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

func (binding *serviceBinding) latencySnapshot(args []runtime.Value) (runtime.Value, error) {
	binding.mu.RLock()
	stats := binding.latency
	binding.mu.RUnlock()

	avgMs := int64(0)
	if stats.count > 0 {
		avgMs = stats.totalMs / stats.count
	}
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"count":   runtime.IntValue{Value: stats.count},
		"totalMs": runtime.IntValue{Value: stats.totalMs},
		"maxMs":   runtime.IntValue{Value: stats.maxMs},
		"avgMs":   runtime.IntValue{Value: avgMs},
	}}, nil
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
	countersValue, err := binding.countersSnapshot(nil)
	if err != nil {
		return nil, err
	}
	latencyValue, err := binding.latencySnapshot(nil)
	if err != nil {
		return nil, err
	}
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"service":        runtime.StringValue{Value: binding.name},
		"startedAt":      runtime.IntValue{Value: binding.startedAt},
		"uptimeMs":       runtime.IntValue{Value: binding.uptimeMs()},
		"health":         healthValue,
		"ready":          readyValue,
		"requestCount":   requestCount,
		"recentRequests": recentRequests,
		"counters":       countersValue,
		"latency":        latencyValue,
	}}, nil
}

func (binding *serviceBinding) uptimeMs() int64 {
	now := time.Now().UnixMilli()
	if now < binding.startedAt {
		return 0
	}
	return now - binding.startedAt
}

func (binding *serviceBinding) normalizeRequestRecord(value runtime.Value) (runtime.Value, int64, int64, error) {
	objectValue, ok := value.(*runtime.ObjectValue)
	if !ok {
		return value, 0, -1, nil
	}

	fields := make(map[string]runtime.Value, len(objectValue.Fields)+1)
	statusCode := int64(0)
	durationMs := int64(-1)
	for key, fieldValue := range objectValue.Fields {
		snapshot, err := observeSnapshotValue(fieldValue)
		if err != nil {
			return nil, 0, -1, err
		}
		fields[key] = snapshot
		if key == "status" {
			intValue, ok := snapshot.(runtime.IntValue)
			if !ok {
				return nil, 0, -1, fmt.Errorf("recordRequest status must be int")
			}
			statusCode = intValue.Value
		}
		if key == "duration_ms" {
			intValue, ok := snapshot.(runtime.IntValue)
			if !ok {
				return nil, 0, -1, fmt.Errorf("recordRequest duration_ms must be int")
			}
			if intValue.Value < 0 {
				return nil, 0, -1, fmt.Errorf("recordRequest duration_ms must be non-negative")
			}
			durationMs = intValue.Value
		}
	}
	if _, ok := fields["ts"]; !ok {
		fields["ts"] = runtime.IntValue{Value: time.Now().UnixMilli()}
	}
	return &runtime.ObjectValue{Fields: fields}, statusCode, durationMs, nil
}

func (binding *serviceBinding) observeLatency(durationMs int64) {
	binding.mu.Lock()
	defer binding.mu.Unlock()
	binding.latency.count++
	binding.latency.totalMs += durationMs
	if durationMs > binding.latency.maxMs {
		binding.latency.maxMs = durationMs
	}
}

func (binding *serviceBinding) incrementCounter(name string, delta int64) int64 {
	binding.mu.Lock()
	defer binding.mu.Unlock()
	binding.counters[name] += delta
	return binding.counters[name]
}

func (binding *serviceBinding) counterValue(name string) int64 {
	binding.mu.RLock()
	defer binding.mu.RUnlock()
	return binding.counters[name]
}

func (binding *serviceBinding) countersObject() *runtime.ObjectValue {
	binding.mu.RLock()
	defer binding.mu.RUnlock()

	fields := make(map[string]runtime.Value, len(binding.counters))
	for key, value := range binding.counters {
		fields[key] = runtime.IntValue{Value: value}
	}
	return &runtime.ObjectValue{Fields: fields}
}
