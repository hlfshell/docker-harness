package redis

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedis(t *testing.T) {
	// Create a new redis container
	r, err := NewRedis(t.Name())
	require.Nil(t, err)
	require.NotNil(t, r)

	// Create the container
	err = r.Create()
	require.Nil(t, err)
	defer r.Cleanup()

	// Ensure that the container is running
	running, err := r.container.IsRunning()
	require.Nil(t, err)
	require.True(t, running)

	// Connect to the database
	client, err := r.ConnectWithTimeout(10 * time.Second)
	require.Nil(t, err)
	require.NotNil(t, client)

	// Ensure that we can ping the database
	pong, err := client.Ping(context.Background()).Result()
	require.Nil(t, err)
	require.Equal(t, "PONG", pong)

	// Ensure we can write a key and read it back
	key := uuid.New().String()
	value := uuid.New().String()
	err = client.Set(context.Background(), key, value, 0).Err()
	require.Nil(t, err)

	result, err := client.Get(context.Background(), key).Result()
	require.Nil(t, err)
	assert.Equal(t, value, result)
}
