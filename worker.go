package payment

import (
	"context"
	"github.com/stripe/stripe-go/v79"
	"sync"

	"go.uber.org/zap"
)

// EventProcessor 定義了處理事件的接口
type EventProcessor interface {
	ProcessEvent(ctx context.Context, event *stripe.Event) error
}

// WorkerPool 管理一組 worker 來處理事件
type WorkerPool struct {
	workers   chan struct{}
	tasks     chan func()
	wg        sync.WaitGroup
	logger    *zap.Logger
	processor EventProcessor
}

// NewWorkerPool 創建一個新的 WorkerPool
func NewWorkerPool(size int, processor EventProcessor, logger *zap.Logger) *WorkerPool {
	wp := &WorkerPool{
		workers:   make(chan struct{}, size),
		tasks:     make(chan func(), 1000), // 緩衝區大小可配置
		logger:    logger,
		processor: processor,
	}

	for i := 0; i < size; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}

	return wp
}

// worker 是處理任務的 goroutine
func (wp *WorkerPool) worker() {
	defer wp.wg.Done()
	for task := range wp.tasks {
		wp.workers <- struct{}{}
		task()
		<-wp.workers
	}
}

// Submit 提交一個事件到 worker pool 進行處理
func (wp *WorkerPool) Submit(ctx context.Context, event *stripe.Event) {
	wp.tasks <- func() {
		if err := wp.processor.ProcessEvent(ctx, event); err != nil {
			wp.logger.Error("Failed to process event",
				zap.Error(err),
				zap.String("event_type", string(event.Type)),
				zap.String("event_id", event.ID))
		}
	}
}

// Shutdown 關閉 worker pool
func (wp *WorkerPool) Shutdown() {
	close(wp.tasks)
	wp.wg.Wait()
}
