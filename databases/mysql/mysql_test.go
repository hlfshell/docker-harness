package mysql

import (
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMysql(t *testing.T) {
	// Create a new mysql container
	m, err := NewMysql(
		t.Name(),
		"",
		"username",
		"password",
		"database",
	)
	require.Nil(t, err)
	require.NotNil(t, m)

	// Create our db container
	err = m.Create()
	require.Nil(t, err)
	defer m.Cleanup()

	// Connect to the database
	db, err := m.ConnectWithTimeout(10 * time.Second)
	require.Nil(t, err)
	require.NotNil(t, db)

	// Ensure our DB is connected
	err = db.Ping()
	require.Nil(t, err)

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
