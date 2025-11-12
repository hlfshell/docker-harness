package memcached

import (
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemcached(t *testing.T) {
	m, err := NewMemcached(t.Name())
	require.Nil(t, err)

	// Create the container
	err = m.Create()
	require.Nil(t, err)
	defer m.Cleanup()

	// Ensure that the container is running
	running, err := m.container.IsRunning()
	require.Nil(t, err)
	require.True(t, running)

	// Connect to the database
	client, err := m.ConnectWithTimeout(10 * time.Second)
	require.Nil(t, err)
	require.NotNil(t, client)

	// Ensure that we can ping the database
	err = client.Ping()
	require.Nil(t, err)

	// Ensure that we can write a key and read it back
	key := uuid.New().String()
	value := uuid.New().String()
	err = client.Set(&memcache.Item{
		Key:        key,
		Value:      []byte(value),
		Expiration: 0,
	})
	require.Nil(t, err)

	result, err := client.Get(key)
	require.Nil(t, err)
	require.NotNil(t, result)
	assert.Equal(t, key, result.Key)
	assert.Equal(t, value, string(result.Value))
}
