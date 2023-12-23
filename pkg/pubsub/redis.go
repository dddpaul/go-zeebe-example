package pubsub

import (
	"github.com/redis/go-redis/v9"
	"golang.org/x/net/context"
)

type RedisPubSub struct {
	rdb redis.UniversalClient
}

func NewRedisPubSub(addr string) PubSub {
	return &RedisPubSub{
		rdb: redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs: []string{addr},
		})}
}

func (p *RedisPubSub) Publish(ctx context.Context, channel string, message interface{}) error {
	if err := p.rdb.Publish(ctx, channel, message).Err(); err != nil {
		return err
	}
	return nil
}

func (p *RedisPubSub) Subscribe(ctx context.Context, channel string) (chan Message, func(ctx context.Context, channel string)) {
	ch := make(chan Message, 1)
	sub := p.rdb.Subscribe(ctx, channel).Channel(redis.WithChannelSize(1))
	go func() {
		ch <- Message{Text: (<-sub).Payload}
	}()
	return ch, nil
}
