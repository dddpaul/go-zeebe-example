package main

import (
	"context"
	"github.com/dddpaul/go-zeebe-example/pkg/pubsub"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRedisPubSub(t *testing.T) {
	channel := "id-" + uuid.NewString()
	message := "Success"
	pubSub := pubsub.NewRedisPubSub()
	ch := pubSub.Subscribe(context.Background(), channel)

	go func() {
		err := pubSub.Publish(context.Background(), channel, message)
		assert.Nil(t, err)
	}()

	received := <-ch
	assert.Equal(t, message, received.(*redis.Message).Payload)
}
