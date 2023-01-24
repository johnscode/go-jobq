package jobq

import (
	"context"
	"github.com/go-redis/redis/v8"
	"time"
)

type JobQProducer struct {
	Redis      *redis.Client
	StreamName string
}

func (p *JobQProducer) SendToQ(ctx context.Context, event map[string]interface{}) error {
	event["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	return p.Redis.XAdd(ctx, &redis.XAddArgs{
		Stream: p.StreamName,
		MaxLen: 0,
		ID:     "",
		Values: event,
	}).Err()
}
