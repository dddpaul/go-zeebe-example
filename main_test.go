package main

import (
	"context"
	"github.com/dddpaul/go-zeebe-example/pkg/pubsub"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRedisPubSub(t *testing.T) {
	channel := "id-" + uuid.NewString()
	message := "Success"
	pubSub := pubsub.NewRedisPubSub(":6379")
	ch := pubSub.Subscribe(context.Background(), channel)

	go func() {
		err := pubSub.Publish(context.Background(), channel, message)
		assert.Nil(t, err)
	}()

	received := <-ch
	assert.Equal(t, message, received.Text)
}
