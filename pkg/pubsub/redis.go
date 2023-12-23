package pubsub

import (
	"github.com/redis/go-redis/v9"
	"golang.org/x/net/context"
)

type PubSub interface {
	Publish(ctx context.Context, channel string, message interface{}) error
	Subscribe(ctx context.Context, channel string) chan interface{}
}

type RedisPubSub struct {
	rdb redis.UniversalClient
}

func NewRedisPubSub() PubSub {
	return &RedisPubSub{
		rdb: redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs: []string{":6379"},
		})}
}

func (p *RedisPubSub) Publish(ctx context.Context, channel string, message interface{}) error {
	if err := p.rdb.Publish(ctx, channel, message).Err(); err != nil {
		return err
	}
	return nil
}

func (p *RedisPubSub) Subscribe(ctx context.Context, channel string) chan interface{} {
	ch := make(chan interface{}, 1)
	sub := p.rdb.Subscribe(ctx, channel).Channel(redis.WithChannelSize(1))
	go func() {
		ch <- <-sub
	}()
	return ch
}
