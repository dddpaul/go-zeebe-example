package pubsub

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	wait "github.com/testcontainers/testcontainers-go/wait"
	"testing"
	"time"
)

func Test_RedisPubSub(t *testing.T) {
	redisAddr, teardown := startTestContainer(t)
	defer teardown()
	pubSub := NewRedisPubSub([]string{redisAddr})

	t.Run("should receive all published message", func(t *testing.T) {
		// given
		channel := "id-" + uuid.NewString()
		messages := []string{"Message-1", "Message-2", "Message-3"}
		ch, cleanup := pubSub.Subscribe(context.Background(), channel)
		defer cleanup(context.Background(), channel)

		// when
		go func() {
			for _, msg := range messages {
				err := pubSub.Publish(context.Background(), channel, msg)
				require.NoError(t, err)
			}
			err := pubSub.Close(context.Background(), channel)
			require.NoError(t, err)
		}()

		// then
		i := 0
		for received := range ch {
			assert.Equal(t, messages[i], received.Text)
			i++
		}
	})

	t.Run("should stop subscribe when context is cancelled", func(t *testing.T) {
		// given
		ctx, cancel := context.WithCancel(context.Background())
		channel := "id-" + uuid.NewString()
		message := "Message-1"
		ch, cleanup := pubSub.Subscribe(ctx, channel)
		defer cleanup(ctx, channel)

		// when
		err := pubSub.Publish(ctx, channel, message)
		require.NoError(t, err)
		ready := make(chan bool)
		go func() {
			<-ready
			cancel()
		}()
		var received string
		for msg := range ch {
			received = msg.Text
			ready <- true
		}

		// then
		assert.Equal(t, message, received)
	})
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
