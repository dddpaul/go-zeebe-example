package pubsub

import (
	"context"
	"github.com/dddpaul/go-zeebe-example/pkg/logger"
	"sync"
)

type SimpleCache struct {
	cache map[string]chan interface{}
	mu    sync.Mutex
}

func NewSimpleCache() PubSub {
	return &SimpleCache{
		cache: make(map[string]chan interface{}),
	}
}

func (c *SimpleCache) Publish(ctx context.Context, channel string, message interface{}) error {
	ch := c.get(channel)
	ch <- message
	logger.Log(ctx, nil).WithField(logger.MESSAGE, message).Debugf("message published")
	return nil
}

func (c *SimpleCache) Subscribe(ctx context.Context, channel string) (chan Message, func(ctx context.Context, channel string)) {
	ch := c.get(channel)
	result := make(chan Message, 1)
	go func() {
		if txt, ok := (<-ch).(string); ok {
			logger.Log(ctx, nil).WithField(logger.MESSAGE, txt).Debugf("message received")
			result <- Message{Text: txt}
		}
	}()
	return result, c.del
}

func (c *SimpleCache) get(channel string) chan interface{} {
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

func (c *SimpleCache) del(ctx context.Context, channel string) {
	c.mu.Lock()
	delete(c.cache, channel)
	defer c.mu.Unlock()
}
