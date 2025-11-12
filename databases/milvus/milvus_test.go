package milvus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	milvus "github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

func TestMilvus(t *testing.T) {
	// Create a new milvus container
	m, err := NewMilvus(
		t.Name(),
		"",
	)
	require.Nil(t, err)
	require.NotNil(t, m)

	// Start the container
	err = m.Create()
	require.Nil(t, err)
	defer m.Cleanup()

	// Ensure that the container is running
	running, err := m.container.IsRunning()
	require.Nil(t, err)
	require.True(t, running)

	// Connect to the database
	client, err := m.ConnectWithTimeout(10)
	require.Nil(t, err)
	require.NotNil(t, client)

	// Create a milvus collection
	collectionName := "test-collection"
	dimension := 128
	indexFileSize := 1024
	metricType := milvus.MetricTypeL2

	collectionParams := milvus.CollectionParam{
		CollectionName: collectionName,
		Dimension:      dimension,
		IndexFileSize:  indexFileSize,
		MetricType:     metricType,
	}
	exists, err := client.HasCollection(context.Background(), collectionName)
	require.Nil(t, err)
	assert.False(t, exists)

	err = client.CreateCollection(context.Background(), &entity.Schema{
		CollectionName: collectionName,
		Fields: []*entity.Field{
			{
				Name:        "id",
				PrimaryKey:  true,
				AutoID:      true,
				Description: "unique id",
				DataType:    entity.FieldTypeInt64,
			},
		},
	}, 128, 1, &milvus.CreateCollectionOption{})
	require.Nil(t, err)

	exists, err = client.HasCollection(context.Background(), collection)
	require.Nil(t, err)
	assert.True(t, exists)
}
