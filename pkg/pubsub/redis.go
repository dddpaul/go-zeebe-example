package pubsub

import (
	"github.com/redis/go-redis/v9"
	"golang.org/x/net/context"
)

type PubSub interface {
	Publish(context.Context, string) error
	Subscribe(ctx context.Context, id string) chan interface{}
}

type RedisPubSub struct {
	rdb *redis.ClusterClient
}

func NewRedisPubSub() PubSub {
	return &RedisPubSub{
		rdb: redis.NewClusterClient(&redis.ClusterOptions{
			Addrs: []string{":6379"},
		})}
}

func (p *RedisPubSub) Publish(ctx context.Context, id string) error {
	if err := p.rdb.Publish(ctx, id, "Success").Err(); err != nil {
		return err
	}
	return nil
}

func (p *RedisPubSub) Subscribe(ctx context.Context, id string) chan interface{} {
	ch := make(chan interface{}, 1)
	go func() {
		for msg := range p.rdb.Subscribe(ctx, id).Channel(redis.WithChannelSize(1)) {
			ch <- msg.String()
		}
	}()
	return ch
}
