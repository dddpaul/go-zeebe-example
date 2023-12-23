package main

import (
	"context"
	"fmt"
	"github.com/dddpaul/go-zeebe-example/pkg/pubsub"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	wait "github.com/testcontainers/testcontainers-go/wait"
	"testing"
	"time"
)

func TestRedisPubSub(t *testing.T) {
	redisAddr, teardown := startTestContainer(t)
	defer teardown()

	channel := "id-" + uuid.NewString()
	message := "Success"
	pubSub := pubsub.NewRedisPubSub([]string{redisAddr})
	ch, _ := pubSub.Subscribe(context.Background(), channel)

	go func() {
		err := pubSub.Publish(context.Background(), channel, message)
		assert.Nil(t, err)
	}()

	received := <-ch
	assert.Equal(t, message, received.Text)
}

func startTestContainer(t *testing.T) (hostAndPort string, teardown func()) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "redis/redis-stack:latest",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForExposedPort().WithStartupTimeout(time.Second * 60),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "6379")
	require.NoError(t, err)

	return fmt.Sprintf("%s:%s", host, port.Port()), func() {
		if err := container.Terminate(ctx); err != nil {
			panic(err)
		}
	}
}
