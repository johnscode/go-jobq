# go-jobq
A Job Queue implementation in golang

This module provides a simple implementation of a job queue. It is suitable for 
situations where an application just needs a basic queuing mechanism to 
handle workloads that can easily be defined using a golang interface{}. 
It provides an in memory queue and a redis backed queue.

The redis-backed queue actually uses an in memory queue to service an incoming
redis stream. the api is slightly different but still pretty simple.

Define a worker to handle the job, pass a factory method for the worker to the q,
then push an interface{} representing the job data to the q. When a worker thread 
becomes available, the interface containing the job data will be passed to your worker.
<!-- TOC -->
* [go-jobq](#go-jobq)
    * [Usage](#usage)
      * [Define the worker and worker factory](#define-the-worker-and-worker-factory)
      * [Add a job to the q](#add-a-job-to-the-q)
    * [Example of an in memory queue](#example-of-an-in-memory-queue)
    * [Example of a Redis backed q](#example-of-a-redis-backed-q)
    * [TODO](#todo)
<!-- TOC -->
### Usage

```shell
go get github.com/johnscode/go-jobq.git
```
To create an in memory job queue first create the queue, then start it:

```go
q := NewQueue(workers int, capacity int, workerFactory)
q.StartWorkers(ctx)
```
Terminate the queue using the Stop method:
```go
q.Stop()
```
#### Define the worker and worker factory
A worker needs to implement the IQueueWorker interface:
```go
type IQueueWorker interface {
  ProcessJob(ctx context.Context, job interface{})
  QuitChan() chan bool
}
```
ProcessJob is the worker function that takes the 'job' data as an interface
and performs your operation.

QuitChan() provides a chan bool that the queue uses to stop the workers.

The queue will use a factory that you provide to create the workers. The 
worker factory implements the QueueWorkerFactory interface:
```go
type QueueWorkerFactory interface {
	CreateWorker(ctx context.Context, quit chan bool) (IQueueWorker, error)
}
```
#### Add a job to the q
```go
q.EnqueueJob(jobdata)
```
For a Redis-backed queue, you need to provide a redis client and a string representing
the name of the redis stream that is the backing store for the queue. Note: you don't need
to create the redis stream, redis will do that for us.
```go
q, _ := CreateRedisStreamJobQueue(redisClient, numWorkers, capacity, workerFactory)
```
### Example of an in memory queue
Create a job q to operate on a series of strings:

a. Define the worker
```go
type struct Worker {
	quitChan chan bool
	// whatever else your worker needs
}
func (w Worker) ProcessJob(ctx context.Context, job interface{}) {
	if str, ok := job.(string); ok {
      // do stuff with the string
    }
}
func (w Worker) QuitChan() chan bool {
	return w.quitChan
}
```
b. Define the factory
```go
type WorkerFactory struct {
// whatever else your factory needs
}
func (f WorkerFactory) CreateWorker(ctx context.Context, quit chan bool) (IQueueWorker, error) {
	return Worker {
	  quitChan: quit	
    }
}
```
c. Create and start the queue with 5 workers and a capacity of 10
```go
q:=NewQueue(5, 10, WorkerFactory{})
// StartWorkers will block until the queue quits (or fails), so use a go thread
go q.StartWorkers(ctx)
```
d. Push a job t the q
```go
q.EnqueueJob("string to process")
```
### Example of a Redis backed q

Create a redis-backed job q to operate on a series of strings:

a. Define the worker and factory (slightly different from above)
```go
type RedisWorker struct {
	quitChan chan bool
}
func (w RedisWorker) ProcessJob(ctx context.Context, job interface{}) {
	if xmsg, ok := job.(redis.XMessage); ok {
		fmt.Printf("process string: %+v\n", xmsg.Values)
	}
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
```
b. Create and start the queue:
```go
rdb, _:= redis.NewClient(...)
quitChan := make(chan bool)
q, _ := CreateRedisStreamJobQueue(rdb, "our-string-queue", 2, 3, WorkerFactory{})
// Work will block until the queue quits (or fails), so use a go thread
// write to quitChan to tell the queue to stop
go q.Work(ctx, quitChan)
```
c. Push data to the redis queue. _Note: this could me in a different process or instance_
```go
q := QProducer{Redis: rdb, "our-string-queue"}
_ = q.SendToQ(ctx, "string to process")
```
### Note on Encoding Payload
For a redis backed queue, the job data needs to be easily marshalled to a string. Redis 
will fail to marshall a data type that needs a binary encoding. Any primitive, struct of
primitives, or map[string]interface{} of primitives will easily marshal. 

To send a struct that has data that requires binary encoding, try marshaling 
the struct to JSON and using the json string as the job data. Similarly, for a byte 
array, try encoding to hex then sending the hex string as the job. Your ProcessJob 
method can easily unmarshal/decode on the other end of the queue.

### TODO
* Replace the factory pattern, seems clunky

