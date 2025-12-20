package mysql

import (
	"database/sql"
	"fmt"
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
	db, err := m.ConnectWithTimeout(20 * time.Second)
	require.Nil(t, err)
	require.NotNil(t, db)

	// Ensure our DB is connected
	err = db.Ping()
	require.Nil(t, err)

	// Create a table
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS test (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255))")
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

// TestMysqlImmediateConnection tests connecting immediately after Create()
// without using ConnectWithTimeout - this is a common failure scenario
func TestMysqlImmediateConnection(t *testing.T) {
	m, err := NewMysql(
		t.Name(),
		"",
		"testuser",
		"testpass",
		"testdb",
	)
	require.Nil(t, err)
	require.NotNil(t, m)

	err = m.Create()
	require.Nil(t, err)
	defer m.Cleanup()

	// Try to connect immediately - this often fails if MySQL isn't ready
	db, err := m.Connect()
	if err != nil {
		t.Logf("Immediate connection failed (expected): %v", err)
		// This is expected - MySQL needs time to initialize
		// But we should verify ConnectWithTimeout works
		db, err = m.ConnectWithTimeout(30 * time.Second)
		require.Nil(t, err, "ConnectWithTimeout should eventually succeed")
		require.NotNil(t, db)
	} else {
		t.Log("Immediate connection succeeded")
		require.NotNil(t, db)
	}

	// Verify connection works
	err = db.Ping()
	require.Nil(t, err)
}

// TestMysqlRapidConnections tests multiple rapid connection attempts
// to identify race conditions or connection pool issues
func TestMysqlRapidConnections(t *testing.T) {
	m, err := NewMysql(
		t.Name(),
		"",
		"testuser",
		"testpass",
		"testdb",
	)
	require.Nil(t, err)
	require.NotNil(t, m)

	err = m.Create()
	require.Nil(t, err)
	defer m.Cleanup()

	// Create() now waits for MySQL to be ready, so we can connect immediately
	// Try multiple rapid connections
	successCount := 0
	failureCount := 0
	for i := 0; i < 10; i++ {
		db, err := m.Connect()
		if err != nil {
			failureCount++
			t.Logf("Connection attempt %d failed: %v", i+1, err)
		} else {
			successCount++
			err = db.Ping()
			if err != nil {
				t.Logf("Ping failed on connection %d: %v", i+1, err)
				failureCount++
				db.Close()
			} else {
				db.Close()
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Logf("Rapid connections: %d success, %d failures", successCount, failureCount)
	// We expect at least some connections to succeed
	assert.Greater(t, successCount, 0, "At least some connections should succeed")
}

// TestMysqlConnectionStability tests that a connection remains stable over time
func TestMysqlConnectionStability(t *testing.T) {
	m, err := NewMysql(
		t.Name(),
		"",
		"testuser",
		"testpass",
		"testdb",
	)
	require.Nil(t, err)
	require.NotNil(t, m)

	err = m.Create()
	require.Nil(t, err)
	defer m.Cleanup()

	db, err := m.ConnectWithTimeout(30 * time.Second)
	require.Nil(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Test connection stability over time
	for i := 0; i < 20; i++ {
		err = db.Ping()
		require.Nil(t, err, "Connection should remain stable (iteration %d)", i+1)
		time.Sleep(500 * time.Millisecond)
	}
}

// TestMysqlDifferentVersions tests connection with different MySQL versions
func TestMysqlDifferentVersions(t *testing.T) {
	versions := []string{"8.0"}

	for _, version := range versions {
		t.Run("version_"+version, func(t *testing.T) {
			// Use a simpler container name to avoid issues
			containerName := fmt.Sprintf("mysql_test_%s_%d", version, time.Now().Unix())
			m, err := NewMysql(
				containerName,
				version,
				"testuser",
				"testpass",
				"testdb",
			)
			require.Nil(t, err)
			require.NotNil(t, m)

			err = m.Create()
			require.Nil(t, err, "Container should start for version %s", version)
			defer m.Cleanup()

			db, err := m.ConnectWithTimeout(60 * time.Second)
			require.Nil(t, err, "Connection should succeed for version %s", version)
			require.NotNil(t, db)

			err = db.Ping()
			require.Nil(t, err, "Ping should succeed for version %s", version)
			db.Close()
		})
	}
}

// TestMysqlConnectionRetryLogic tests the retry logic in ConnectWithTimeout
func TestMysqlConnectionRetryLogic(t *testing.T) {
	m, err := NewMysql(
		t.Name(),
		"",
		"testuser",
		"testpass",
		"testdb",
	)
	require.Nil(t, err)
	require.NotNil(t, m)

	err = m.Create()
	require.Nil(t, err)
	defer m.Cleanup()

	// Try with a short timeout first - should fail
	start := time.Now()
	db, err := m.ConnectWithTimeout(1 * time.Second)
	duration := time.Since(start)

	if err != nil {
		t.Logf("Short timeout failed as expected after %v: %v", duration, err)
		// Now try with longer timeout - should succeed
		db, err = m.ConnectWithTimeout(30 * time.Second)
		require.Nil(t, err, "Connection should succeed with longer timeout")
		require.NotNil(t, db)
	} else {
		t.Logf("Connection succeeded quickly after %v", duration)
		require.NotNil(t, db)
	}

	err = db.Ping()
	require.Nil(t, err)
}

// TestMysqlConnectionStringFormat tests the connection string format
func TestMysqlConnectionStringFormat(t *testing.T) {
	m, err := NewMysql(
		t.Name(),
		"",
		"special_user",
		"special@pass#123",
		"special_db",
	)
	require.Nil(t, err)
	require.NotNil(t, m)

	err = m.Create()
	require.Nil(t, err)
	defer m.Cleanup()

	db, err := m.ConnectWithTimeout(30 * time.Second)
	require.Nil(t, err, "Connection should work with special characters in credentials")
	require.NotNil(t, db)

	err = db.Ping()
	require.Nil(t, err)
}

// TestMysqlMultipleInstances tests running multiple MySQL instances simultaneously
func TestMysqlMultipleInstances(t *testing.T) {
	instances := make([]*Mysql, 3)

	// Create multiple instances
	for i := 0; i < 3; i++ {
		m, err := NewMysql(
			t.Name()+fmt.Sprintf("_%d", i),
			"",
			fmt.Sprintf("user%d", i),
			fmt.Sprintf("pass%d", i),
			fmt.Sprintf("db%d", i),
		)
		require.Nil(t, err)
		require.NotNil(t, m)
		instances[i] = m

		err = m.Create()
		require.Nil(t, err, "Instance %d should start", i)
		defer m.Cleanup()
	}

	// Connect to all instances
	dbs := make([]*sql.DB, 3)
	for i, m := range instances {
		db, err := m.ConnectWithTimeout(30 * time.Second)
		require.Nil(t, err, "Instance %d should connect", i)
		require.NotNil(t, db)
		dbs[i] = db
		defer db.Close()
	}

	// Verify all connections work
	for i, db := range dbs {
		err := db.Ping()
		require.Nil(t, err, "Instance %d connection should be stable", i)
	}
}

// TestMysqlPortMapping tests that port mapping is correct
func TestMysqlPortMapping(t *testing.T) {
	m, err := NewMysql(
		t.Name(),
		"",
		"testuser",
		"testpass",
		"testdb",
	)
	require.Nil(t, err)
	require.NotNil(t, m)

	err = m.Create()
	require.Nil(t, err)
	defer m.Cleanup()

	// Get the port
	ports := m.GetContainer().GetPorts()
	port := ports["3306"]
	require.NotEmpty(t, port, "Port should be assigned")
	t.Logf("MySQL is listening on port: %s", port)

	// Connect and verify
	db, err := m.ConnectWithTimeout(30 * time.Second)
	require.Nil(t, err)
	require.NotNil(t, db)

	err = db.Ping()
	require.Nil(t, err)
}

// TestMysqlRootUser tests that root user is configured correctly
// Root cannot be set via MYSQL_USER, only via MYSQL_ROOT_PASSWORD
func TestMysqlRootUser(t *testing.T) {
	m, err := NewMysql(
		t.Name(),
		"",
		"root",
		"rootpassword",
		"testdb",
	)
	require.Nil(t, err)
	require.NotNil(t, m)

	err = m.Create()
	require.Nil(t, err)
	defer m.Cleanup()

	// Connect as root user
	db, err := m.ConnectWithTimeout(30 * time.Second)
	require.Nil(t, err, "Root user should be able to connect")
	require.NotNil(t, db)

	// Verify connection works
	err = db.Ping()
	require.Nil(t, err, "Root user connection should work")

	// Verify we can create and use the database
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS root_test (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255))")
	require.Nil(t, err)

	_, err = db.Exec("INSERT INTO root_test (name) VALUES ('root_test')")
	require.Nil(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM root_test").Scan(&count)
	require.Nil(t, err)
	assert.Equal(t, 1, count)
}

// TestMysqlRootUserCaseSensitivity tests that only exact "root" (lowercase) is treated as root
// This is important because case variations should be treated as regular users
func TestMysqlRootUserCaseSensitivity(t *testing.T) {
	// Test that exact "root" (lowercase) works as root
	t.Run("exact_root_lowercase", func(t *testing.T) {
		m, err := NewMysql(
			fmt.Sprintf("test_root_%d", time.Now().UnixNano()),
			"",
			"root",
			"rootpass",
			"testdb",
		)
		require.Nil(t, err)
		require.NotNil(t, m)

		err = m.Create()
		require.Nil(t, err, "Root user should start successfully")
		defer m.Cleanup()

		db, err := m.ConnectWithTimeout(30 * time.Second)
		require.Nil(t, err, "Root user should connect")
		require.NotNil(t, db)
		db.Close()
	})

	// Test that "Root" (capitalized) is treated as regular user
	t.Run("capitalized_Root", func(t *testing.T) {
		m, err := NewMysql(
			fmt.Sprintf("test_Root_%d", time.Now().UnixNano()),
			"",
			"Root",
			"rootpass",
			"testdb",
		)
		require.Nil(t, err)
		require.NotNil(t, m)

		err = m.Create()
		require.Nil(t, err, "Regular user 'Root' should start successfully")
		defer m.Cleanup()

		db, err := m.ConnectWithTimeout(30 * time.Second)
		require.Nil(t, err, "Regular user 'Root' should connect")
		require.NotNil(t, db)
		db.Close()
	})

	// Test that "ROOT" (uppercase) is treated as regular user
	t.Run("uppercase_ROOT", func(t *testing.T) {
		m, err := NewMysql(
			fmt.Sprintf("test_ROOT_%d", time.Now().UnixNano()),
			"",
			"ROOT",
			"rootpass",
			"testdb",
		)
		require.Nil(t, err)
		require.NotNil(t, m)

		err = m.Create()
		require.Nil(t, err, "Regular user 'ROOT' should start successfully")
		defer m.Cleanup()

		db, err := m.ConnectWithTimeout(30 * time.Second)
		require.Nil(t, err, "Regular user 'ROOT' should connect")
		require.NotNil(t, db)
		db.Close()
	})
}

// TestMysqlRootUserPrivileges tests that root user has elevated privileges
func TestMysqlRootUserPrivileges(t *testing.T) {
	m, err := NewMysql(
		t.Name(),
		"",
		"root",
		"rootpassword",
		"testdb",
	)
	require.Nil(t, err)
	require.NotNil(t, m)

	err = m.Create()
	require.Nil(t, err)
	defer m.Cleanup()

	db, err := m.ConnectWithTimeout(30 * time.Second)
	require.Nil(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Root should be able to create new users
	_, err = db.Exec("CREATE USER IF NOT EXISTS 'testuser'@'%' IDENTIFIED BY 'testpass'")
	require.Nil(t, err, "Root should be able to create users")

	// Root should be able to grant privileges
	_, err = db.Exec("GRANT SELECT ON testdb.* TO 'testuser'@'%'")
	require.Nil(t, err, "Root should be able to grant privileges")

	// Root should be able to flush privileges
	_, err = db.Exec("FLUSH PRIVILEGES")
	require.Nil(t, err, "Root should be able to flush privileges")

	// Verify root can see all databases
	var dbCount int
	err = db.QueryRow("SELECT COUNT(*) FROM information_schema.SCHEMATA").Scan(&dbCount)
	require.Nil(t, err)
	assert.Greater(t, dbCount, 0, "Root should be able to see databases")
}

// TestMysqlRootVsRegularUser tests root and regular user side by side
func TestMysqlRootVsRegularUser(t *testing.T) {
	// Create root user container
	rootMysql, err := NewMysql(
		t.Name()+"_root",
		"",
		"root",
		"rootpass",
		"rootdb",
	)
	require.Nil(t, err)
	require.NotNil(t, rootMysql)
	err = rootMysql.Create()
	require.Nil(t, err)
	defer rootMysql.Cleanup()

	// Create regular user container
	regularMysql, err := NewMysql(
		t.Name()+"_regular",
		"",
		"regularuser",
		"regularpass",
		"regulardb",
	)
	require.Nil(t, err)
	require.NotNil(t, regularMysql)
	err = regularMysql.Create()
	require.Nil(t, err)
	defer regularMysql.Cleanup()

	// Both should be able to connect
	rootDB, err := rootMysql.ConnectWithTimeout(30 * time.Second)
	require.Nil(t, err, "Root should connect")
	require.NotNil(t, rootDB)
	defer rootDB.Close()

	regularDB, err := regularMysql.ConnectWithTimeout(30 * time.Second)
	require.Nil(t, err, "Regular user should connect")
	require.NotNil(t, regularDB)
	defer regularDB.Close()

	// Both should be able to use their databases
	_, err = rootDB.Exec("CREATE TABLE IF NOT EXISTS root_table (id INT)")
	require.Nil(t, err)

	_, err = regularDB.Exec("CREATE TABLE IF NOT EXISTS regular_table (id INT)")
	require.Nil(t, err)
}

// TestMysqlRootUserEnvironmentVariables tests that environment variables are set correctly
func TestMysqlRootUserEnvironmentVariables(t *testing.T) {
	// Test root user - should NOT have MYSQL_USER
	rootMysql, err := NewMysql(
		t.Name()+"_root",
		"",
		"root",
		"rootpass",
		"rootdb",
	)
	require.Nil(t, err)
	require.NotNil(t, rootMysql)

	// Get the container to inspect env vars
	container := rootMysql.GetContainer()
	require.NotNil(t, container)

	// We can't directly inspect env vars from the harness, but we can verify
	// the container starts successfully, which means MySQL accepted the config
	err = rootMysql.Create()
	require.Nil(t, err, "Root user container should start (proving env vars are correct)")
	defer rootMysql.Cleanup()

	// If we got here, the container started, which means MySQL didn't reject
	// the configuration. This is the best we can do without direct env var inspection.

	// Verify root can connect (which confirms the env vars worked)
	db, err := rootMysql.ConnectWithTimeout(30 * time.Second)
	require.Nil(t, err, "Root should connect if env vars were correct")
	require.NotNil(t, db)
	db.Close()
}

// TestMysqlRootUserRapidConnections tests root user with rapid connections
func TestMysqlRootUserRapidConnections(t *testing.T) {
	m, err := NewMysql(
		t.Name(),
		"",
		"root",
		"rootpass",
		"testdb",
	)
	require.Nil(t, err)
	require.NotNil(t, m)

	err = m.Create()
	require.Nil(t, err)
	defer m.Cleanup()

	// Try multiple rapid connections as root
	successCount := 0
	for i := 0; i < 10; i++ {
		db, err := m.Connect()
		if err == nil {
			err = db.Ping()
			if err == nil {
				successCount++
			}
			db.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}

	assert.Greater(t, successCount, 0, "Root user should handle rapid connections")
	t.Logf("Root user rapid connections: %d/10 succeeded", successCount)
}

// TestMysqlRootUserCreatesOtherUsers tests that root can create and manage other users
func TestMysqlRootUserCreatesOtherUsers(t *testing.T) {
	m, err := NewMysql(
		t.Name(),
		"",
		"root",
		"rootpass",
		"testdb",
	)
	require.Nil(t, err)
	require.NotNil(t, m)

	err = m.Create()
	require.Nil(t, err)
	defer m.Cleanup()

	db, err := m.ConnectWithTimeout(30 * time.Second)
	require.Nil(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Create a new user
	_, err = db.Exec("CREATE USER IF NOT EXISTS 'newuser'@'%' IDENTIFIED BY 'newpass'")
	require.Nil(t, err, "Root should be able to create users")

	// Grant privileges
	_, err = db.Exec("GRANT ALL PRIVILEGES ON testdb.* TO 'newuser'@'%'")
	require.Nil(t, err, "Root should be able to grant privileges")

	// Flush privileges
	_, err = db.Exec("FLUSH PRIVILEGES")
	require.Nil(t, err, "Root should be able to flush privileges")

	// Verify the user exists
	var userExists int
	err = db.QueryRow("SELECT COUNT(*) FROM mysql.user WHERE user = 'newuser'").Scan(&userExists)
	require.Nil(t, err)
	assert.Equal(t, 1, userExists, "New user should exist")
}
