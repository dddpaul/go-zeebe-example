package pubsub

import (
	"golang.org/x/net/context"
)

type PubSub interface {
	Publish(ctx context.Context, channel string, message interface{}) error
	Subscribe(ctx context.Context, channel string) chan Message
}

type Message struct {
	Text string
}
