package jobq

import (
	"context"
	"log"
	"sync"
)

// IQueueWorker
type IQueueWorker interface {
	ProcessJob(ctx context.Context, job interface{})
	QuitChan() chan bool
}

// QueueWorkerFactory
type QueueWorkerFactory interface {
	CreateWorker(ctx context.Context, quit chan bool) (IQueueWorker, error)
}

// Queue
type Queue struct {
	WorkerCount   int
	Capacity      int
	JobChan       chan interface{}
	Wg            *sync.WaitGroup
	QuitChans     []chan bool
	WorkerFactory QueueWorkerFactory
}

// NewQueue
func NewQueue(workers int, capacity int, workerFactory QueueWorkerFactory) Queue {
	var wg sync.WaitGroup
	jobQueue := make(chan interface{}, capacity)
	quitChans := make([]chan bool, workers)
	return Queue{
		WorkerCount:   workers,
		JobChan:       jobQueue,
		WorkerFactory: workerFactory,
		Wg:            &wg,
		QuitChans:     quitChans,
	}
}

// Stop
func (q *Queue) Stop() {
	for i := range q.QuitChans {
		q.QuitChans[i] <- true
	}
}

// EnqueueJobNoBlock
func (q *Queue) EnqueueJobNoBlock(job interface{}) bool {
	select {
	case q.JobChan <- job:
		return true
	default:
		return false
	}
}

// EnqueueJob
func (q *Queue) EnqueueJob(job interface{}) {
	q.JobChan <- job
}

// StartWorkers
// call from a go routine since it blocks until all work done
func (q *Queue) StartWorkers(ctx context.Context) {
	for i := 0; i < q.WorkerCount; i++ {
		q.Wg.Add(1)
		quitChan := make(chan bool)
		q.QuitChans[i] = quitChan
		worker, err := q.WorkerFactory.CreateWorker(ctx, quitChan)
		if err == nil {
			go q.work(ctx, worker)
		} else {
			log.Fatalln("error creating worker")
		}
	}
	q.Wg.Wait()
}

// work
func (q *Queue) work(ctx context.Context, worker IQueueWorker) {
	defer q.Wg.Done()
	for {
		select {
		case <-worker.QuitChan():
			return
		case job := <-q.JobChan:
			worker.ProcessJob(ctx, job)
		}
	}
}
