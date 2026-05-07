package concurrency

import (
	"context"
	"errors"
	"testing"
	"time"

	langruntime "icoo_lang/internal/runtime"
)

func TestPoolRejectsWhenQueueIsFull(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	pool := NewPool(1, 1, func(task *GoTask) {
		select {
		case <-started:
		default:
			close(started)
		}
		<-release
	})

	if err := pool.Submit(langruntime.IntValue{Value: 1}, nil); err != nil {
		t.Fatalf("first submit failed: %v", err)
	}

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start in time")
	}

	if err := pool.Submit(langruntime.IntValue{Value: 2}, nil); err != nil {
		t.Fatalf("second submit failed: %v", err)
	}

	if err := pool.Submit(langruntime.IntValue{Value: 3}, nil); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("expected ErrQueueFull, got %v", err)
	}

	stats := pool.Stats()
	if stats.Rejected != 1 {
		t.Fatalf("expected 1 rejected task, got %d", stats.Rejected)
	}
	if stats.QueueCapacity != 1 {
		t.Fatalf("expected queue capacity 1, got %d", stats.QueueCapacity)
	}

	close(release)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := pool.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
}

func TestPoolRejectsSubmitAfterShutdown(t *testing.T) {
	pool := NewPool(1, 1, func(task *GoTask) {})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := pool.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	if err := pool.Submit(langruntime.IntValue{Value: 1}, nil); !errors.Is(err, ErrPoolClosed) {
		t.Fatalf("expected ErrPoolClosed, got %v", err)
	}
}
