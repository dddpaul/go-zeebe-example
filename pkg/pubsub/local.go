package pubsub

import (
	"context"
	"sync"

	"github.com/dddpaul/go-zeebe-example/pkg/logger"
)

type LocalPubSub struct {
	cache map[string]chan interface{}
	mu    sync.Mutex
}

func NewLocalPubSub() PubSub {
	return &LocalPubSub{
		cache: make(map[string]chan interface{}),
	}
}

func (p *LocalPubSub) Publish(ctx context.Context, channel string, message interface{}) error {
	ch := p.get(channel)
	ch <- message
	logger.Log(ctx, nil).WithField(logger.MESSAGE, message).Debugf("message published")
	return nil
}

func (p *LocalPubSub) Subscribe(ctx context.Context, channel string) (chan Message, func(ctx context.Context, channel string)) {
	ch := p.get(channel)
	result := make(chan Message, 1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(result)
				return
			case msg, ok := <-ch:
				if !ok {
					close(result)
					return
				}
				if txt, ok := msg.(string); ok {
					logger.Log(ctx, nil).WithField(logger.MESSAGE, txt).Debugf("message received")
					result <- Message{Text: txt}
				}
			}
		}
	}()
	return result, p.del
}

func (p *LocalPubSub) Close(ctx context.Context, channel string) error {
	ch := p.get(channel)
	close(ch)
	return nil
}

func (p *LocalPubSub) get(channel string) chan interface{} {
	var result chan interface{}
	p.mu.Lock()
	if ch, ok := p.cache[channel]; ok {
		result = ch
	} else {
		result = make(chan interface{}, 1)
		p.cache[channel] = result
	}
	defer p.mu.Unlock()
	return result
}

func (p *LocalPubSub) del(ctx context.Context, channel string) {
	p.mu.Lock()
	delete(p.cache, channel)
	defer p.mu.Unlock()
}
