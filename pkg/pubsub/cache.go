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
	ch := c.newChannel(channel)
	ch <- message
	logger.Log(ctx, nil).WithField(logger.MESSAGE, message).Debugf("message published")
	return nil
}

func (c *SimpleCache) Subscribe(ctx context.Context, channel string) chan Message {
	ch := c.newChannel(channel)
	ch1 := make(chan Message, 1)
	go func() {
		if txt, ok := (<-ch).(string); ok {
			logger.Log(ctx, nil).WithField(logger.MESSAGE, txt).Debugf("message received")
			ch1 <- Message{Text: txt}
		}
	}()
	return ch1
}

func (c *SimpleCache) newChannel(channel string) chan interface{} {
	var ch chan interface{}
	c.mu.Lock()
	if ch1, ok := c.cache[channel]; ok {
		ch = ch1
	} else {
		ch = make(chan interface{}, 1)
		c.cache[channel] = ch
	}
	c.mu.Unlock()
	return ch
}
