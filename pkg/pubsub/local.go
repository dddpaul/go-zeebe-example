package pubsub

import (
	"context"
	"github.com/dddpaul/go-zeebe-example/pkg/logger"
	"sync"
)

type localPubSub struct {
	cache map[string]chan interface{}
	mu    sync.Mutex
}

func NewLocalPubSub() PubSub {
	return &localPubSub{
		cache: make(map[string]chan interface{}),
	}
}

func (c *localPubSub) Publish(ctx context.Context, channel string, message interface{}) error {
	ch := c.get(channel)
	ch <- message
	logger.Log(ctx, nil).WithField(logger.MESSAGE, message).Debugf("message published")
	return nil
}

func (c *localPubSub) Subscribe(ctx context.Context, channel string) (chan Message, func(ctx context.Context, channel string)) {
	ch := c.get(channel)
	result := make(chan Message, 1)
	go func() {
		for msg := range ch {
			if txt, ok := msg.(string); ok {
				logger.Log(ctx, nil).WithField(logger.MESSAGE, txt).Debugf("message received")
				result <- Message{Text: txt}
			}
		}
		close(result)
	}()
	return result, c.del
}

func (c *localPubSub) Close(ctx context.Context, channel string) error {
	ch := c.get(channel)
	close(ch)
	return nil
}

func (c *localPubSub) get(channel string) chan interface{} {
	var result chan interface{}
	c.mu.Lock()
	if ch, ok := c.cache[channel]; ok {
		result = ch
	} else {
		result = make(chan interface{}, 1)
		c.cache[channel] = result
	}
	defer c.mu.Unlock()
	return result
}

func (c *localPubSub) del(ctx context.Context, channel string) {
	c.mu.Lock()
	delete(c.cache, channel)
	defer c.mu.Unlock()
}
