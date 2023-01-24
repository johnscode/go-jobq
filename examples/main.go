package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/johnscode/go-jobq"
	"time"
)

type Worker struct {
	quitChan chan bool
}

func (w Worker) ProcessJob(ctx context.Context, job interface{}) {
	if str, ok := job.(string); ok {
		fmt.Printf("process string: %s\n", str)
	}
}
func (w Worker) QuitChan() chan bool {
	return w.quitChan
}

type WorkerFactory struct {
	// whatever else your factory needs
}

func (f WorkerFactory) CreateWorker(ctx context.Context, quit chan bool) (jobq.IQueueWorker, error) {
	return Worker{
		quitChan: quit,
	}, nil
}

type RedisWorker struct {
	quitChan chan bool
}

func (w RedisWorker) ProcessJob(ctx context.Context, job interface{}) {
	if xmsg, ok := job.(redis.XMessage); ok {
		fmt.Printf("process string: %+v\n", xmsg.Values)
	}
	//fmt.Printf("fffff")
}
func (w RedisWorker) QuitChan() chan bool {
	return w.quitChan
}

type RedisWorkerFactory struct {
	// whatever else your factory needs
}

func (f RedisWorkerFactory) CreateWorker(ctx context.Context, quit chan bool) (jobq.IQueueWorker, error) {
	return RedisWorker{
		quitChan: quit,
	}, nil
}

func main() {
	q := jobq.NewQueue(2, 5, WorkerFactory{})
	go q.StartWorkers(context.Background())
	q.EnqueueJob("string to process")
	time.Sleep(time.Second)
	q.Stop()

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", "localhost", 6379),
		Password: "",
		DB:       0, // use default DB
	})
	// redisq is the 'consumer' side of the q
	nm := "streamq"
	quitChan := make(chan bool, 1)
	rq := jobq.RedisStreamQueue{Redis: rdb, Queue: jobq.NewQueue(2, 3, RedisWorkerFactory{}),
		StreamName: nm,
	}
	go func() {
		rq.Work(context.Background(), quitChan)
	}()
	//redisq := jobq.CreateRedisStreamJobQueue(rdb, "our-string-queue", 1, 10, WorkerFactory{})
	//quitChan := make(chan bool, 1)
	//go redisq.Work(context.Background(), quitChan)
	// 'producer' side of the q
	producer := jobq.JobQProducer{Redis: rdb, StreamName: nm}
	err := producer.SendToQ(context.Background(), map[string]interface{}{"job": "string to process"})
	if err != nil {
		fmt.Printf("err %+v", err)
	}
	time.Sleep(5 * time.Second)
	quitChan <- true
}
