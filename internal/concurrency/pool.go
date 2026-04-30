package concurrency

import (
	"fmt"
	"sync"

	"icoo_lang/internal/runtime"
)

type GoTask struct {
	Callee runtime.Value
	Args   []runtime.Value
}

type TaskExecutor func(task *GoTask)

type GoroutinePool struct {
	tasks   chan *GoTask
	stop    chan struct{}
	wg      sync.WaitGroup
	Workers int
	exec    TaskExecutor
}

func NewPool(workers int, executor TaskExecutor) *GoroutinePool {
	if workers <= 0 {
		workers = 4
	}
	pool := &GoroutinePool{
		tasks:   make(chan *GoTask, workers*16),
		stop:    make(chan struct{}),
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
	for {
		select {
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("panic in goroutine pool: %v\n", r)
					}
				}()
				p.exec(task)
			}()
		case <-p.stop:
			return
		}
	}
}

func (p *GoroutinePool) Submit(callee runtime.Value, args []runtime.Value) {
	if p == nil {
		return
	}
	task := &GoTask{Callee: callee, Args: args}
	select {
	case p.tasks <- task:
	default:
		go func() { p.tasks <- task }()
	}
}

func (p *GoroutinePool) Shutdown() {
	close(p.stop)
	p.wg.Wait()
}
