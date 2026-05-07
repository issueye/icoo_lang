package core

import (
	"fmt"
	"sync"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"
)

// observeRecentBinding 观测最近值的绑定结构
type observeRecentBinding struct {
	mu    sync.RWMutex
	limit int
	total int64
	items []runtime.Value
}

// LoadStdCoreObserveModule 加载 std.core.observe 模块
func LoadStdCoreObserveModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.core.observe",
		Path: "std.core.observe",
		Exports: map[string]runtime.Value{
			"recent": &runtime.NativeFunction{Name: "recent", Arity: 1, Fn: observeRecent},
		},
		Done: true,
	}
}

// observeRecent 创建最近值观测器
func observeRecent(args []runtime.Value) (runtime.Value, error) {
	limitValue, ok := args[0].(runtime.IntValue)
	if !ok {
		return nil, fmt.Errorf("recent expects integer limit")
	}
	if limitValue.Value < 0 {
		return nil, fmt.Errorf("recent expects non-negative limit")
	}

	binding := &observeRecentBinding{
		limit: int(limitValue.Value),
		items: []runtime.Value{},
	}
	return binding.object(), nil
}

// object 返回观测器对象
func (binding *observeRecentBinding) object() *runtime.ObjectValue {
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"add":   &runtime.NativeFunction{Name: "observe.recent.add", Arity: 1, Fn: binding.add},
		"clear": &runtime.NativeFunction{Name: "observe.recent.clear", Arity: 0, Fn: binding.clear},
		"count": &runtime.NativeFunction{Name: "observe.recent.count", Arity: 0, Fn: binding.count},
		"list":  &runtime.NativeFunction{Name: "observe.recent.list", Arity: 0, Fn: binding.list},
		"total": &runtime.NativeFunction{Name: "observe.recent.total", Arity: 0, Fn: binding.totalCount},
	}}
}

// add 添加值到观测器
func (binding *observeRecentBinding) add(args []runtime.Value) (runtime.Value, error) {
	snapshot, err := observeSnapshotValue(args[0])
	if err != nil {
		return nil, err
	}

	binding.mu.Lock()
	defer binding.mu.Unlock()

	binding.total++
	if binding.limit == 0 {
		return runtime.NullValue{}, nil
	}

	binding.items = append([]runtime.Value{snapshot}, binding.items...)
	if len(binding.items) > binding.limit {
		binding.items = binding.items[:binding.limit]
	}
	return runtime.NullValue{}, nil
}

// clear 清空观测器
func (binding *observeRecentBinding) clear(args []runtime.Value) (runtime.Value, error) {
	binding.mu.Lock()
	defer binding.mu.Unlock()
	binding.items = binding.items[:0]
	return runtime.NullValue{}, nil
}

// count 获取当前数量
func (binding *observeRecentBinding) count(args []runtime.Value) (runtime.Value, error) {
	binding.mu.RLock()
	defer binding.mu.RUnlock()
	return runtime.IntValue{Value: int64(len(binding.items))}, nil
}

// list 获取所有值列表
func (binding *observeRecentBinding) list(args []runtime.Value) (runtime.Value, error) {
	binding.mu.RLock()
	defer binding.mu.RUnlock()

	items := make([]runtime.Value, 0, len(binding.items))
	for _, item := range binding.items {
		snapshot, err := observeSnapshotValue(item)
		if err != nil {
			return nil, err
		}
		items = append(items, snapshot)
	}
	return &runtime.ArrayValue{Elements: items}, nil
}

// totalCount 获取总计数
func (binding *observeRecentBinding) totalCount(args []runtime.Value) (runtime.Value, error) {
	binding.mu.RLock()
	defer binding.mu.RUnlock()
	return runtime.IntValue{Value: binding.total}, nil
}

// observeSnapshotValue 创建值的快照
func observeSnapshotValue(value runtime.Value) (runtime.Value, error) {
	plain, err := utils.RuntimeToPlainValue(value)
	if err != nil {
		return nil, err
	}
	return utils.PlainToRuntimeValue(plain), nil
}
