package jobq

import (
	"context"
	"github.com/go-redis/redis/v8"
	"log"
)

// RedisStreamQueue
type RedisStreamQueue struct {
	Redis      *redis.Client
	Queue      Queue
	StreamName string
}

// Work
// start from a go thread
func (q RedisStreamQueue) Work(ctx context.Context, quitChan chan bool) {
	go q.Queue.StartWorkers(ctx)
	messageID := "0"
	for {
		select {
		case <-quitChan:
			q.Queue.Stop()
			return
		default:
			messageID = processRedisStream(q, messageID)
		}
	}
}

// processRedisStream
func processRedisStream(q RedisStreamQueue, messageID string) string {
	var ctx = context.Background()
	msgId := messageID
	data, err := q.Redis.XRead(ctx, &redis.XReadArgs{
		Streams: []string{q.StreamName, msgId},
		// number of entries to retrieve, note that there is a problem if
		// we try to pull multiple entries here. entries get repeatedly
		// processed in this case; as if not being timely deleted from redis stream
		Count: 1,
		Block: 0,
	}).Result()
	if err != nil {
		log.Printf("error reading redis stream")
		// what to do here with failed read, abort worker?
	} else {
		for _, eventStream := range data {
			for _, message := range eventStream.Messages {
				q.Queue.EnqueueJob(message)
				msgId = message.ID
				if err := q.Redis.XDel(context.Background(), q.StreamName, message.ID).Err(); err != nil {
					log.Printf("error deleting redis msg")
				}
			}
		}
	}
	return msgId
}

// Enqueue
func (q RedisStreamQueue) Enqueue(job map[string]interface{}) error {
	return q.Redis.XAdd(context.Background(), &redis.XAddArgs{
		Stream: q.StreamName,
		MaxLen: 0,
		ID:     "",
		Values: job,
	}).Err()
}

// CreateRedisStreamJobQueue
func CreateRedisStreamJobQueue(redis *redis.Client, streamName string, workerCount int, capacity int, workerFactory QueueWorkerFactory) RedisStreamQueue {
	jq := RedisStreamQueue{
		Redis:      redis,
		Queue:      NewQueue(workerCount, capacity, workerFactory),
		StreamName: streamName,
	}
	return jq
}
