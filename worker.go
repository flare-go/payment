package payment

import (
	"context"
	"github.com/stripe/stripe-go/v79"
	"go.uber.org/zap"
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
				w.stripePayment.logger.Info("開始處理事件",
					zap.String("event_type", string(job.Event.Type)),
					zap.String("event_id", job.Event.ID))

				err := w.stripePayment.processEvent(job.Ctx, job.Event)

				if err != nil {
					w.stripePayment.logger.Error("處理事件時發生錯誤",
						zap.Error(err),
						zap.String("event_type", string(job.Event.Type)),
						zap.String("event_id", job.Event.ID))
				} else {
					w.stripePayment.logger.Info("事件處理完成",
						zap.String("event_type", string(job.Event.Type)),
						zap.String("event_id", job.Event.ID))
				}

			case <-w.quit:
				return
			}
		}
	}()
}

func (w Worker) Stop() {
	close(w.quit)
}
