package vm

import (
	goruntime "runtime"

	"icoo_lang/internal/concurrency"
	"icoo_lang/internal/runtime"
)

type MemoryStats struct {
	AllocBytes        uint64
	TotalAllocBytes   uint64
	SysBytes          uint64
	HeapAllocBytes    uint64
	HeapSysBytes      uint64
	HeapIdleBytes     uint64
	HeapInuseBytes    uint64
	HeapReleasedBytes uint64
	HeapObjects       uint64
	NumGC             uint32
}

type RuntimeStats struct {
	NumCPU       int
	NumGoroutine int
	Memory       MemoryStats
	Pool         concurrency.PoolStats
}

func (vm *VM) Stats() RuntimeStats {
	var mem goruntime.MemStats
	goruntime.ReadMemStats(&mem)

	poolStats := concurrency.PoolStats{}
	vm.mu.RLock()
	if vm.pool != nil {
		poolStats = vm.pool.Stats()
	}
	vm.mu.RUnlock()

	return RuntimeStats{
		NumCPU:       goruntime.NumCPU(),
		NumGoroutine: goruntime.NumGoroutine(),
		Memory: MemoryStats{
			AllocBytes:        mem.Alloc,
			TotalAllocBytes:   mem.TotalAlloc,
			SysBytes:          mem.Sys,
			HeapAllocBytes:    mem.HeapAlloc,
			HeapSysBytes:      mem.HeapSys,
			HeapIdleBytes:     mem.HeapIdle,
			HeapInuseBytes:    mem.HeapInuse,
			HeapReleasedBytes: mem.HeapReleased,
			HeapObjects:       mem.HeapObjects,
			NumGC:             mem.NumGC,
		},
		Pool: poolStats,
	}
}

func (vm *VM) CollectGarbage() RuntimeStats {
	goruntime.GC()
	return vm.Stats()
}

func (vm *VM) runtimeStatsValue() runtime.Value {
	return runtimeStatsToValue(vm.Stats())
}

func (vm *VM) collectGarbageValue() runtime.Value {
	return runtimeStatsToValue(vm.CollectGarbage())
}

func runtimeStatsToValue(stats RuntimeStats) runtime.Value {
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"numCPU":       runtime.IntValue{Value: int64(stats.NumCPU)},
		"goroutines":   runtime.IntValue{Value: int64(stats.NumGoroutine)},
		"memory":       memoryStatsToValue(stats.Memory),
		"goroutinePool": poolStatsToValue(stats.Pool),
	}}
}

func memoryStatsToValue(stats MemoryStats) runtime.Value {
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"allocBytes":        runtime.IntValue{Value: int64(stats.AllocBytes)},
		"totalAllocBytes":   runtime.IntValue{Value: int64(stats.TotalAllocBytes)},
		"sysBytes":          runtime.IntValue{Value: int64(stats.SysBytes)},
		"heapAllocBytes":    runtime.IntValue{Value: int64(stats.HeapAllocBytes)},
		"heapSysBytes":      runtime.IntValue{Value: int64(stats.HeapSysBytes)},
		"heapIdleBytes":     runtime.IntValue{Value: int64(stats.HeapIdleBytes)},
		"heapInuseBytes":    runtime.IntValue{Value: int64(stats.HeapInuseBytes)},
		"heapReleasedBytes": runtime.IntValue{Value: int64(stats.HeapReleasedBytes)},
		"heapObjects":       runtime.IntValue{Value: int64(stats.HeapObjects)},
		"numGC":             runtime.IntValue{Value: int64(stats.NumGC)},
	}}
}

func poolStatsToValue(stats concurrency.PoolStats) runtime.Value {
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"workers":       runtime.IntValue{Value: int64(stats.Workers)},
		"queueCapacity": runtime.IntValue{Value: int64(stats.QueueCapacity)},
		"queued":        runtime.IntValue{Value: int64(stats.Queued)},
		"active":        runtime.IntValue{Value: stats.Active},
		"submitted":     runtime.IntValue{Value: int64(stats.Submitted)},
		"completed":     runtime.IntValue{Value: int64(stats.Completed)},
		"rejected":      runtime.IntValue{Value: int64(stats.Rejected)},
		"panics":        runtime.IntValue{Value: int64(stats.Panics)},
		"closed":        runtime.BoolValue{Value: stats.Closed},
	}}
}
