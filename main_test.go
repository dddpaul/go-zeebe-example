package main

import (
	"context"
	"github.com/dddpaul/go-zeebe-example/pkg/pubsub"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRedisPubSub(t *testing.T) {
	id := "id-123"
	//message := "Success"
	ps := pubsub.NewRedisPubSub()
	//ch := ps.Subscribe(context.Background(), id)

	//go func() {
	err := ps.Publish(context.Background(), id)
	assert.Nil(t, err)
	//}()

	//for msg := range ch {
	//	fmt.Println(msg)
	//	assert.Equal(t, message, msg)
	//}
}
