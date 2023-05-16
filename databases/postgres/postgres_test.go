package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgres(t *testing.T) {
	// Create a new postgres container
	p, err := NewPostgres(
		t.Name(),
		"",
		"username",
		"password",
		"database",
	)
	require.Nil(t, err)
	require.NotNil(t, p)

	// Connect to the database
	db, err := p.Create()
	require.Nil(t, err)
	require.NotNil(t, db)
	defer p.Cleanup()

	// Create a table
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS test (id SERIAL PRIMARY KEY, name TEXT)")
	require.Nil(t, err)

	// Insert a row
	_, err = db.Exec("INSERT INTO test (name) VALUES ('test')")
	require.Nil(t, err)

	// Query the row
	rows, err := db.Query("SELECT * FROM test")
	require.Nil(t, err)
	require.NotNil(t, rows)

	// Ensure that the row is there
	count := 0
	for rows.Next() {
		count++
	}
	assert.Equal(t, 1, count)
}
