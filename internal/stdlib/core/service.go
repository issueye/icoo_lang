package core

import (
	"fmt"
	"sync"
	"time"

	"icoo_lang/internal/runtime"
)

// serviceBinding 服务绑定结构
type serviceBinding struct {
	mu          sync.RWMutex
	name        string
	startedAt   int64
	ready       bool
	readyReason string
	counters    map[string]int64
	latency     serviceLatencyStats
	recent      *observeRecentBinding
	events      *observeRecentBinding
	logger      runtime.Value
	reloader    runtime.Value
	suppliers   map[string]runtime.Value
}

// serviceLatencyStats 延迟统计结构
type serviceLatencyStats struct {
	count   int64
	totalMs int64
	maxMs   int64
}

// LoadStdCoreServiceModule 加载 std.core.service 模块
func LoadStdCoreServiceModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.core.service",
		Path: "std.core.service",
		Exports: map[string]runtime.Value{
			"create": &runtime.NativeFunction{Name: "create", Arity: 1, Fn: serviceCreate},
		},
		Done: true,
	}
}

// serviceCreate 创建服务对象
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

	eventLimit := recentLimit
	if eventLimitValue, ok := options.Fields["eventLimit"]; ok {
		intValue, ok := eventLimitValue.(runtime.IntValue)
		if !ok {
			return nil, fmt.Errorf("create eventLimit must be int")
		}
		if intValue.Value < 0 {
			return nil, fmt.Errorf("create eventLimit must be non-negative")
		}
		eventLimit = intValue.Value
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
		events: &observeRecentBinding{
			limit: int(eventLimit),
			items: []runtime.Value{},
		},
		suppliers: map[string]runtime.Value{},
	}
	return binding.object(), nil
}

// object 返回服务对象
func (binding *serviceBinding) object() *runtime.ObjectValue {
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"name":              runtime.StringValue{Value: binding.name},
		"startedAt":         runtime.IntValue{Value: binding.startedAt},
		"health":            &runtime.NativeFunction{Name: "service.health", Arity: 0, Fn: binding.health},
		"ready":             &runtime.NativeFunction{Name: "service.ready", Arity: 0, Fn: binding.readyStatus},
		"markReady":         &runtime.NativeFunction{Name: "service.markReady", Arity: 0, Fn: binding.markReady},
		"markNotReady":      &runtime.NativeFunction{Name: "service.markNotReady", Arity: -1, Fn: binding.markNotReady},
		"increment":         &runtime.NativeFunction{Name: "service.increment", Arity: -1, Fn: binding.increment},
		"counter":           &runtime.NativeFunction{Name: "service.counter", Arity: 1, Fn: binding.counter},
		"counters":          &runtime.NativeFunction{Name: "service.counters", Arity: 0, Fn: binding.countersSnapshot},
		"setLogger":         &runtime.NativeFunction{Name: "service.setLogger", Arity: 1, Fn: binding.setLogger},
		"log":               &runtime.NativeFunction{Name: "service.log", Arity: -1, CtxFn: binding.log},
		"recordEvent":       &runtime.NativeFunction{Name: "service.recordEvent", Arity: 1, Fn: binding.recordEvent},
		"clearEvents":       &runtime.NativeFunction{Name: "service.clearEvents", Arity: 0, Fn: binding.clearEvents},
		"eventCount":        &runtime.NativeFunction{Name: "service.eventCount", Arity: 0, Fn: binding.eventCount},
		"recentEvents":      &runtime.NativeFunction{Name: "service.recentEvents", Arity: 0, Fn: binding.recentEvents},
		"setReload":         &runtime.NativeFunction{Name: "service.setReload", Arity: 1, Fn: binding.setReload},
		"reload":            &runtime.NativeFunction{Name: "service.reload", Arity: 0, CtxFn: binding.reload},
		"setSupplierHealth": &runtime.NativeFunction{Name: "service.setSupplierHealth", Arity: -1, Fn: binding.setSupplierHealth},
		"supplierHealth":    &runtime.NativeFunction{Name: "service.supplierHealth", Arity: 1, Fn: binding.supplierHealth},
		"suppliersHealth":   &runtime.NativeFunction{Name: "service.suppliersHealth", Arity: 0, Fn: binding.suppliersHealth},
		"recordRequest":     &runtime.NativeFunction{Name: "service.recordRequest", Arity: 1, Fn: binding.recordRequest},
		"clearRequests":     &runtime.NativeFunction{Name: "service.clearRequests", Arity: 0, Fn: binding.clearRequests},
		"requestCount":      &runtime.NativeFunction{Name: "service.requestCount", Arity: 0, Fn: binding.requestCount},
		"recentRequests":    &runtime.NativeFunction{Name: "service.recentRequests", Arity: 0, Fn: binding.recentRequests},
		"latency":           &runtime.NativeFunction{Name: "service.latency", Arity: 0, Fn: binding.latencySnapshot},
		"snapshot":          &runtime.NativeFunction{Name: "service.snapshot", Arity: 0, Fn: binding.snapshot},
	}}
}

// health 返回健康状态
func (binding *serviceBinding) health(args []runtime.Value) (runtime.Value, error) {
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"ok":        runtime.BoolValue{Value: true},
		"service":   runtime.StringValue{Value: binding.name},
		"startedAt": runtime.IntValue{Value: binding.startedAt},
		"uptimeMs":  runtime.IntValue{Value: binding.uptimeMs()},
	}}, nil
}

// readyStatus 返回就绪状态
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

// markReady 标记为就绪
func (binding *serviceBinding) markReady(args []runtime.Value) (runtime.Value, error) {
	binding.mu.Lock()
	defer binding.mu.Unlock()
	binding.ready = true
	binding.readyReason = ""
	return runtime.NullValue{}, nil
}

// markNotReady 标记为未就绪
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

// increment 增加计数器
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

// counter 获取计数器值
func (binding *serviceBinding) counter(args []runtime.Value) (runtime.Value, error) {
	nameValue, ok := args[0].(runtime.StringValue)
	if !ok || nameValue.Value == "" {
		return nil, fmt.Errorf("counter expects non-empty string name")
	}
	return runtime.IntValue{Value: binding.counterValue(nameValue.Value)}, nil
}

// countersSnapshot 获取所有计数器快照
func (binding *serviceBinding) countersSnapshot(args []runtime.Value) (runtime.Value, error) {
	return binding.countersObject(), nil
}

// setLogger 设置日志处理器
func (binding *serviceBinding) setLogger(args []runtime.Value) (runtime.Value, error) {
	binding.mu.Lock()
	defer binding.mu.Unlock()
	switch value := args[0].(type) {
	case runtime.NullValue:
		binding.logger = nil
	case *runtime.Closure, *runtime.NativeFunction:
		binding.logger = value
	default:
		return nil, fmt.Errorf("setLogger expects callable or null")
	}
	return runtime.NullValue{}, nil
}

// log 记录日志
func (binding *serviceBinding) log(ctx *runtime.NativeContext, args []runtime.Value) (runtime.Value, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("log expects level, message, and optional fields")
	}
	level, err := requireServiceStringArg("log", args[0])
	if err != nil {
		return nil, err
	}
	message, err := requireServiceStringArg("log", args[1])
	if err != nil {
		return nil, err
	}
	event := &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"kind":    runtime.StringValue{Value: "log"},
		"level":   runtime.StringValue{Value: level},
		"message": runtime.StringValue{Value: message},
		"service": runtime.StringValue{Value: binding.name},
		"ts":      runtime.IntValue{Value: time.Now().UnixMilli()},
	}}
	if len(args) == 3 {
		fields, ok := args[2].(*runtime.ObjectValue)
		if !ok {
			return nil, fmt.Errorf("log fields must be object")
		}
		event.Fields["fields"] = fields
	}
	if _, err := binding.recordEvent([]runtime.Value{event}); err != nil {
		return nil, err
	}
	binding.mu.RLock()
	logger := binding.logger
	binding.mu.RUnlock()
	if logger != nil && ctx != nil {
		if _, err := ctx.CallDetached(logger, []runtime.Value{event}); err != nil {
			return nil, err
		}
	}
	return event, nil
}

// recordEvent 记录事件
func (binding *serviceBinding) recordEvent(args []runtime.Value) (runtime.Value, error) {
	eventValue, err := binding.normalizeEventRecord(args[0])
	if err != nil {
		return nil, err
	}
	return binding.events.add([]runtime.Value{eventValue})
}

// clearEvents 清空事件
func (binding *serviceBinding) clearEvents(args []runtime.Value) (runtime.Value, error) {
	return binding.events.clear(args)
}

// eventCount 获取事件数量
func (binding *serviceBinding) eventCount(args []runtime.Value) (runtime.Value, error) {
	return binding.events.totalCount(args)
}

// recentEvents 获取最近事件
func (binding *serviceBinding) recentEvents(args []runtime.Value) (runtime.Value, error) {
	return binding.events.list(args)
}

// setReload 设置重载处理器
func (binding *serviceBinding) setReload(args []runtime.Value) (runtime.Value, error) {
	binding.mu.Lock()
	defer binding.mu.Unlock()
	switch value := args[0].(type) {
	case runtime.NullValue:
		binding.reloader = nil
	case *runtime.Closure, *runtime.NativeFunction:
		binding.reloader = value
	default:
		return nil, fmt.Errorf("setReload expects callable or null")
	}
	return runtime.NullValue{}, nil
}

// reload 执行重载
func (binding *serviceBinding) reload(ctx *runtime.NativeContext, args []runtime.Value) (runtime.Value, error) {
	binding.mu.RLock()
	reloader := binding.reloader
	binding.mu.RUnlock()
	if reloader == nil {
		return &runtime.ObjectValue{Fields: map[string]runtime.Value{
			"ok":      runtime.BoolValue{Value: false},
			"changed": runtime.BoolValue{Value: false},
			"reason":  runtime.StringValue{Value: "reload handler is not configured"},
		}}, nil
	}
	result, err := ctx.CallDetached(reloader, nil)
	if err != nil {
		return nil, err
	}
	reloadEvent := &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"kind":    runtime.StringValue{Value: "reload"},
		"service": runtime.StringValue{Value: binding.name},
		"ts":      runtime.IntValue{Value: time.Now().UnixMilli()},
	}}
	if result != nil {
		reloadEvent.Fields["result"] = result
	}
	if _, err := binding.recordEvent([]runtime.Value{reloadEvent}); err != nil {
		return nil, err
	}
	return result, nil
}

// setSupplierHealth 设置供应商健康状态
func (binding *serviceBinding) setSupplierHealth(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("setSupplierHealth expects name, status, and optional detail")
	}
	name, err := requireServiceStringArg("setSupplierHealth", args[0])
	if err != nil {
		return nil, err
	}
	status, err := binding.normalizeSupplierHealth(name, args[1], args[2:])
	if err != nil {
		return nil, err
	}
	binding.mu.Lock()
	binding.suppliers[name] = status
	binding.mu.Unlock()
	return status, nil
}

// supplierHealth 获取供应商健康状态
func (binding *serviceBinding) supplierHealth(args []runtime.Value) (runtime.Value, error) {
	name, err := requireServiceStringArg("supplierHealth", args[0])
	if err != nil {
		return nil, err
	}
	binding.mu.RLock()
	defer binding.mu.RUnlock()
	if value, ok := binding.suppliers[name]; ok {
		return value, nil
	}
	return runtime.NullValue{}, nil
}

// suppliersHealth 获取所有供应商健康状态
func (binding *serviceBinding) suppliersHealth(args []runtime.Value) (runtime.Value, error) {
	binding.mu.RLock()
	defer binding.mu.RUnlock()
	fields := make(map[string]runtime.Value, len(binding.suppliers))
	for key, value := range binding.suppliers {
		fields[key] = value
	}
	return &runtime.ObjectValue{Fields: fields}, nil
}

// recordRequest 记录请求
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

// clearRequests 清空请求记录
func (binding *serviceBinding) clearRequests(args []runtime.Value) (runtime.Value, error) {
	return binding.recent.clear(args)
}

// requestCount 获取请求数量
func (binding *serviceBinding) requestCount(args []runtime.Value) (runtime.Value, error) {
	return binding.recent.totalCount(args)
}

// recentRequests 获取最近请求
func (binding *serviceBinding) recentRequests(args []runtime.Value) (runtime.Value, error) {
	return binding.recent.list(args)
}

// latencySnapshot 获取延迟统计快照
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

// snapshot 获取完整快照
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
	eventCount, err := binding.eventCount(nil)
	if err != nil {
		return nil, err
	}
	recentEvents, err := binding.recentEvents(nil)
	if err != nil {
		return nil, err
	}
	suppliers, err := binding.suppliersHealth(nil)
	if err != nil {
		return nil, err
	}
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"service":        runtime.StringValue{Value: binding.name},
		"startedAt":      runtime.IntValue{Value: binding.startedAt},
		"uptimeMs":       runtime.IntValue{Value: binding.uptimeMs()},
		"health":         healthValue,
		"ready":          readyValue,
		"eventCount":     eventCount,
		"recentEvents":   recentEvents,
		"suppliers":      suppliers,
		"requestCount":   requestCount,
		"recentRequests": recentRequests,
		"counters":       countersValue,
		"latency":        latencyValue,
	}}, nil
}

// uptimeMs 获取运行时间（毫秒）
func (binding *serviceBinding) uptimeMs() int64 {
	now := time.Now().UnixMilli()
	if now < binding.startedAt {
		return 0
	}
	return now - binding.startedAt
}

// normalizeRequestRecord 规范化请求记录
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

// normalizeEventRecord 规范化事件记录
func (binding *serviceBinding) normalizeEventRecord(value runtime.Value) (runtime.Value, error) {
	objectValue, ok := value.(*runtime.ObjectValue)
	if !ok {
		snapshot, err := observeSnapshotValue(value)
		if err != nil {
			return nil, err
		}
		return &runtime.ObjectValue{Fields: map[string]runtime.Value{
			"value": snapshot,
			"ts":    runtime.IntValue{Value: time.Now().UnixMilli()},
		}}, nil
	}
	fields := make(map[string]runtime.Value, len(objectValue.Fields)+1)
	for key, fieldValue := range objectValue.Fields {
		snapshot, err := observeSnapshotValue(fieldValue)
		if err != nil {
			return nil, err
		}
		fields[key] = snapshot
	}
	if _, ok := fields["ts"]; !ok {
		fields["ts"] = runtime.IntValue{Value: time.Now().UnixMilli()}
	}
	return &runtime.ObjectValue{Fields: fields}, nil
}

// normalizeSupplierHealth 规范化供应商健康状态
func (binding *serviceBinding) normalizeSupplierHealth(name string, status runtime.Value, extra []runtime.Value) (runtime.Value, error) {
	fields := map[string]runtime.Value{
		"name":      runtime.StringValue{Value: name},
		"checkedAt": runtime.IntValue{Value: time.Now().UnixMilli()},
	}
	switch value := status.(type) {
	case runtime.BoolValue:
		fields["ok"] = value
	case *runtime.ObjectValue:
		for key, item := range value.Fields {
			snapshot, err := observeSnapshotValue(item)
			if err != nil {
				return nil, err
			}
			fields[key] = snapshot
		}
		if _, ok := fields["ok"]; !ok {
			return nil, fmt.Errorf("setSupplierHealth object status requires ok field")
		}
	default:
		return nil, fmt.Errorf("setSupplierHealth status must be bool or object")
	}
	if len(extra) == 1 {
		switch value := extra[0].(type) {
		case runtime.StringValue:
			fields["reason"] = value
		case *runtime.ObjectValue:
			fields["meta"] = value
		default:
			return nil, fmt.Errorf("setSupplierHealth detail must be string or object")
		}
	}
	return &runtime.ObjectValue{Fields: fields}, nil
}

// observeLatency 记录延迟观测
func (binding *serviceBinding) observeLatency(durationMs int64) {
	binding.mu.Lock()
	defer binding.mu.Unlock()
	binding.latency.count++
	binding.latency.totalMs += durationMs
	if durationMs > binding.latency.maxMs {
		binding.latency.maxMs = durationMs
	}
}

// incrementCounter 增加计数器
func (binding *serviceBinding) incrementCounter(name string, delta int64) int64 {
	binding.mu.Lock()
	defer binding.mu.Unlock()
	binding.counters[name] += delta
	return binding.counters[name]
}

// counterValue 获取计数器值
func (binding *serviceBinding) counterValue(name string) int64 {
	binding.mu.RLock()
	defer binding.mu.RUnlock()
	return binding.counters[name]
}

// countersObject 获取计数器对象
func (binding *serviceBinding) countersObject() *runtime.ObjectValue {
	binding.mu.RLock()
	defer binding.mu.RUnlock()

	fields := make(map[string]runtime.Value, len(binding.counters))
	for key, value := range binding.counters {
		fields[key] = runtime.IntValue{Value: value}
	}
	return &runtime.ObjectValue{Fields: fields}
}

// requireServiceStringArg 要求服务字符串参数
func requireServiceStringArg(name string, value runtime.Value) (string, error) {
	text, ok := value.(runtime.StringValue)
	if !ok || text.Value == "" {
		return "", fmt.Errorf("%s expects non-empty string argument", name)
	}
	return text.Value, nil
}
