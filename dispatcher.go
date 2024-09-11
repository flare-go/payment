package payment

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	minTickerInterval = 5 * time.Second
	maxTickerInterval = 30 * time.Second
)

type Dispatcher struct {
	WorkerPool    chan chan WorkRequest
	maxWorkers    int
	jobQueue      chan WorkRequest
	stripePayment *StripePayment
	workers       []Worker
	stop          chan bool
	mu            sync.Mutex
}

func NewDispatcher(maxWorkers int, jobQueueSize int, stripePayment *StripePayment) *Dispatcher {
	pool := make(chan chan WorkRequest, maxWorkers)
	return &Dispatcher{
		WorkerPool:    pool,
		maxWorkers:    maxWorkers,
		jobQueue:      make(chan WorkRequest, jobQueueSize),
		stripePayment: stripePayment,
		stop:          make(chan bool),
	}
}

func (d *Dispatcher) Run() {
	for i := 0; i < d.maxWorkers; i++ {
		worker := NewWorker(i+1, d.WorkerPool, d.stripePayment)
		worker.Start()
		d.workers = append(d.workers, worker)
	}

	go d.dispatch()
}

func (d *Dispatcher) dispatch() {
	tickerInterval := 10 * time.Second
	ticker := time.NewTicker(tickerInterval)
	defer ticker.Stop()
	var wg sync.WaitGroup

	for {
		select {
		case job := <-d.jobQueue:
			wg.Add(1)
			go func(job WorkRequest) {
				defer wg.Done()
				select {
				case jobChannel := <-d.WorkerPool:
					select {
					case jobChannel <- job:
						// 成功將任務發送給 worker
					case <-job.Ctx.Done():
						d.stripePayment.logger.Warn("Job context canceled before processing",
							zap.Error(job.Ctx.Err()),
							zap.String("event_type", string(job.Event.Type)),
							zap.String("event_id", job.Event.ID))
					}
				case <-job.Ctx.Done():
					d.stripePayment.logger.Warn("Job context canceled while waiting for available worker",
						zap.Error(job.Ctx.Err()),
						zap.String("event_type", string(job.Event.Type)),
						zap.String("event_id", job.Event.ID))
				}
			}(job)

		case <-ticker.C:
			d.adjustWorkerPool()

			jobQueueLength := len(d.jobQueue)
			if jobQueueLength > 50 {
				tickerInterval = minTickerInterval
			} else if jobQueueLength > 20 {
				tickerInterval = 10 * time.Second
			} else {
				tickerInterval = maxTickerInterval
			}

			ticker.Reset(tickerInterval)
		case <-d.stop:
			wg.Wait()
			return
		}
	}
}

func (d *Dispatcher) adjustWorkerPool() {
	d.mu.Lock()
	defer d.mu.Unlock()

	threshold := float64(len(d.jobQueue)) * 0.75
	currentWorkerCount := len(d.workers)

	if float64(len(d.jobQueue)) > threshold && currentWorkerCount < d.maxWorkers {
		newWorker := NewWorker(currentWorkerCount+1, d.WorkerPool, d.stripePayment)
		newWorker.Start()
		d.workers = append(d.workers, newWorker)
		d.stripePayment.logger.Info("Added new worker", zap.Int("worker_id", newWorker.ID))
	}

	if float64(len(d.jobQueue)) < threshold/2 && currentWorkerCount > 1 {
		worker := d.workers[len(d.workers)-1]
		worker.Stop()
		d.workers = d.workers[:len(d.workers)-1]
		d.stripePayment.logger.Info("Removed worker", zap.Int("worker_id", worker.ID))
	}

	d.cleanupStoppedWorkers()

	if len(d.jobQueue) > 0 && len(d.workers) == 0 {
		newWorker := NewWorker(1, d.WorkerPool, d.stripePayment)
		newWorker.Start()
		d.workers = append(d.workers, newWorker)
		d.stripePayment.logger.Info("Added a new worker because job queue is not empty but no workers are available")
	}
}

func (d *Dispatcher) cleanupStoppedWorkers() {
	var activeWorkers []Worker
	for _, worker := range d.workers {
		select {
		case <-worker.quit:
			d.stripePayment.logger.Info("Cleaned up stopped worker", zap.Int("worker_id", worker.ID))
		default:
			activeWorkers = append(activeWorkers, worker)
		}
	}
	d.workers = activeWorkers
}

func (d *Dispatcher) Stop() {
	close(d.stop)
	var wg sync.WaitGroup

	d.mu.Lock()
	for _, worker := range d.workers {
		wg.Add(1)
		go func(w Worker) {
			defer wg.Done()
			w.Stop()
		}(worker)
	}
	d.mu.Unlock()

	wg.Wait()
}
