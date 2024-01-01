package pubsub

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_LocalPubSub(t *testing.T) {
	pubSub := NewLocalPubSub()

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
