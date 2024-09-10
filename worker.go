package payment

import (
	"context"
	"go.uber.org/zap"
	"time"

	"github.com/stripe/stripe-go/v79"
)

type Worker struct {
	ID            int
	WorkerPool    chan chan WorkRequest
	JobChannel    chan WorkRequest
	quit          chan bool
	stripePayment *StripePayment
}

type WorkRequest struct {
	Event *stripe.Event
	Ctx   context.Context
}

func NewWorker(id int, workerPool chan chan WorkRequest, stripePayment *StripePayment) Worker {
	return Worker{
		ID:            id,
		WorkerPool:    workerPool,
		JobChannel:    make(chan WorkRequest),
		quit:          make(chan bool),
		stripePayment: stripePayment,
	}
}

func (w Worker) Start() {
	go func() {
		for {
			w.WorkerPool <- w.JobChannel

			select {
			case job := <-w.JobChannel:
				if err := w.stripePayment.processEvent(job.Ctx, job.Event); err != nil {
					w.stripePayment.logger.Error("Error processing event",
						zap.Error(err),
						zap.String("event_type", string(job.Event.Type)),
						zap.String("event_id", job.Event.ID))
				}
			case <-time.After(5 * time.Minute): // 設置5分鐘的超時
				w.stripePayment.logger.Info("Worker timed out due to inactivity", zap.Int("worker_id", w.ID))
				w.Stop() // 停止該 Worker
				return
			case <-w.quit:
				return
			}
		}
	}()
}

func (w Worker) Stop() {
	close(w.quit) // 關閉 quit 通道，確保 Worker 正常停止
}
