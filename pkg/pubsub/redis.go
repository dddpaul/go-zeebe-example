package pubsub

import (
	"github.com/redis/go-redis/v9"
	"golang.org/x/net/context"
)

type RedisPubSub struct {
	rdb    redis.UniversalClient
	pubSub *redis.PubSub
}

func NewRedisPubSub(addrs []string) PubSub {
	return &RedisPubSub{
		rdb: redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs: addrs,
		})}
}

func (p *RedisPubSub) Publish(ctx context.Context, channel string, message interface{}) error {
	if err := p.rdb.Publish(ctx, channel, message).Err(); err != nil {
		return err
	}
	return nil
}

func (p *RedisPubSub) Subscribe(ctx context.Context, channel string) (chan Message, func(ctx context.Context, channel string)) {
	size := 1
	result := make(chan Message, size)
	p.pubSub = p.rdb.Subscribe(ctx, channel)
	ch := p.pubSub.Channel(redis.WithChannelSize(size))
	go func() {
		for msg := range ch {
			result <- Message{Text: msg.Payload}
		}
		close(result)
	}()
	return result, nil
}

func (p *RedisPubSub) Close() {
	//if err := p.rdb.Close(); err != nil {
	//	panic(err)
	//}
	if p.pubSub != nil {
		if err := p.pubSub.Close(); err != nil {
			panic(err)
		}
	}
}
