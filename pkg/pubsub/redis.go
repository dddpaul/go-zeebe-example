package pubsub

import (
	"github.com/redis/go-redis/v9"
	"golang.org/x/net/context"
	"sync"
)

type RedisPubSub struct {
	rdb   redis.UniversalClient
	cache map[string]*redis.PubSub
	mu    sync.Mutex
}

func NewRedisPubSub(addrs []string) PubSub {
	return &RedisPubSub{
		rdb: redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs: addrs,
		}),
		cache: make(map[string]*redis.PubSub),
	}
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
	ch := p.get(ctx, channel).Channel(redis.WithChannelSize(size))
	go func() {
		for msg := range ch {
			result <- Message{Text: msg.Payload}
		}
		close(result)
	}()
	return result, p.del
}

func (p *RedisPubSub) Close(ctx context.Context, channel string) error {
	//if err := p.rdb.Close(); err != nil {
	//	panic(err)
	//}
	if err := p.get(ctx, channel).Close(); err != nil {
		return err
	}
	return nil
}

func (p *RedisPubSub) get(ctx context.Context, channel string) *redis.PubSub {
	var result *redis.PubSub
	p.mu.Lock()
	if pubSub, ok := p.cache[channel]; ok {
		result = pubSub
	} else {
		result = p.rdb.Subscribe(ctx, channel)
		p.cache[channel] = result
	}
	defer p.mu.Unlock()
	return result
}

func (p *RedisPubSub) del(ctx context.Context, channel string) {
	p.mu.Lock()
	delete(p.cache, channel)
	defer p.mu.Unlock()
}
