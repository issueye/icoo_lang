package concurrency

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"icoo_lang/internal/runtime"
)

var (
	ErrPoolClosed = errors.New("goroutine pool is shut down")
	ErrQueueFull  = errors.New("goroutine pool queue is full")
)

type GoTask struct {
	Callee runtime.Value
	Args   []runtime.Value
}

type TaskExecutor func(task *GoTask)

type PoolStats struct {
	Workers       int
	QueueCapacity int
	Queued        int
	Active        int64
	Submitted     uint64
	Completed     uint64
	Rejected      uint64
	Panics        uint64
	Closed        bool
}

type GoroutinePool struct {
	tasks   chan *GoTask
	wg      sync.WaitGroup
	Workers int
	exec    TaskExecutor

	mu     sync.RWMutex
	closed bool

	active    atomic.Int64
	submitted atomic.Uint64
	completed atomic.Uint64
	rejected  atomic.Uint64
	panics    atomic.Uint64
}

func NewPool(workers, queueSize int, executor TaskExecutor) *GoroutinePool {
	if workers <= 0 {
		workers = 4
	}
	if queueSize <= 0 {
		queueSize = workers * 16
	}
	pool := &GoroutinePool{
		tasks:   make(chan *GoTask, queueSize),
		Workers: workers,
		exec:    executor,
	}
	pool.start()
	return pool
}

func (p *GoroutinePool) start() {
	for i := 0; i < p.Workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

func (p *GoroutinePool) worker() {
	defer p.wg.Done()
	for task := range p.tasks {
		p.active.Add(1)
		func() {
			defer func() {
				p.active.Add(-1)
				p.completed.Add(1)
				if r := recover(); r != nil {
					p.panics.Add(1)
					fmt.Printf("panic in goroutine pool: %v\n", r)
				}
			}()
			p.exec(task)
		}()
	}
}

func (p *GoroutinePool) Submit(callee runtime.Value, args []runtime.Value) error {
	if p == nil {
		return nil
	}
	task := &GoTask{Callee: callee, Args: args}

	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		p.rejected.Add(1)
		return ErrPoolClosed
	}

	select {
	case p.tasks <- task:
		p.submitted.Add(1)
		return nil
	default:
		p.rejected.Add(1)
		return ErrQueueFull
	}
}

func (p *GoroutinePool) Shutdown(ctx context.Context) error {
	if p == nil {
		return nil
	}

	p.mu.Lock()
	if !p.closed {
		p.closed = true
		close(p.tasks)
	}
	p.mu.Unlock()

	done := make(chan struct{})
	go func() {
		defer close(done)
		p.wg.Wait()
	}()

	if ctx == nil {
		<-done
		return nil
	}

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *GoroutinePool) Stats() PoolStats {
	if p == nil {
		return PoolStats{}
	}
	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()
	return PoolStats{
		Workers:       p.Workers,
		QueueCapacity: cap(p.tasks),
		Queued:        len(p.tasks),
		Active:        p.active.Load(),
		Submitted:     p.submitted.Load(),
		Completed:     p.completed.Load(),
		Rejected:      p.rejected.Load(),
		Panics:        p.panics.Load(),
		Closed:        closed,
	}
}
