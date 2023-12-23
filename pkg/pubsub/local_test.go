package pubsub

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
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
}
