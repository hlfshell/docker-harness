package mysql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	harness "github.com/hlfshell/docker-harness"
)

func init() {
	// Suppress MySQL driver error logging during connection retries
	mysql.SetLogger(&mysql.NopLogger{})
}

type Mysql struct {
	db        *sql.DB
	container *harness.Container
	username  string
	password  string
	database  string
	port      string
}

func NewMysql(name string, tag string, username string, password string, database string) (*Mysql, error) {
	container, err := harness.NewContainer(
		name,
		"mysql",
		tag,
		map[string]string{
			"3306": "",
		},
		map[string]string{
			"MYSQL_USER":          username,
			"MYSQL_PASSWORD":      password,
			"MYSQL_ROOT_PASSWORD": password,
			"MYSQL_DATABASE":      database,
		},
	)
	if err != nil {
		return nil, err
	}
	return &Mysql{
		container: container,
		username:  username,
		password:  password,
		database:  database,
	}, nil
}

func (m *Mysql) Create() error {
	err := m.container.Start()
	if err != nil {
		return err
	}

	// Grab the assigned port
	ports := m.container.GetPorts()
	m.port = ports["3306"]

	// Wait for container to be running
	start := time.Now()
	timeout := 10 * time.Second
	running := false
	for !running && time.Since(start) < timeout {
		running, err = m.container.IsRunning()
		if err != nil {
			m.container.Cleanup()
			return err
		}
		if !running {
			time.Sleep(100 * time.Millisecond)
		}
	}
	if !running {
		m.container.Cleanup()
		return fmt.Errorf("container failed to start within timeout")
	}

	// Wait for MySQL to be ready by attempting connections
	// MySQL can take 10-30 seconds to fully initialize, especially on slower systems
	connectionString := fmt.Sprintf(
		"%s:%s@tcp(localhost:%s)/%s",
		m.username,
		m.password,
		m.port,
		m.database,
	)

	readyTimeout := 60 * time.Second
	readyStart := time.Now()
	for time.Since(readyStart) < readyTimeout {
		db, err := sql.Open("mysql", connectionString)
		if err == nil {
			err = db.Ping()
			if err == nil {
				// MySQL is ready!
				db.Close()
				return nil
			}
			db.Close()
		}
		// Wait before retrying
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("MySQL failed to become ready within %v", readyTimeout)
}

func (m *Mysql) Connect() (*sql.DB, error) {
	connectionString := fmt.Sprintf(
		"%s:%s@tcp(localhost:%s)/%s",
		m.username,
		m.password,
		m.port,
		m.database,
	)

	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		return nil, err
	}

	// Configure connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Ping the database to ensure it is ready
	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	// Close previous connection if it exists
	if m.db != nil {
		m.db.Close()
	}
	m.db = db

	return db, nil
}

func (m *Mysql) ConnectWithTimeout(timeout time.Duration) (*sql.DB, error) {
	start := time.Now()
	var lastErr error

	for time.Since(start) < timeout {
		db, err := m.Connect()
		if err == nil && db != nil {
			// Verify the connection is actually working
			if pingErr := db.Ping(); pingErr == nil {
				return db, nil
			}
			// If ping fails, close and try again
			db.Close()
			if m.db == db {
				m.db = nil
			}
			lastErr = fmt.Errorf("connection opened but ping failed: %w", err)
		} else {
			lastErr = err
		}
		// Wait between retries to avoid overwhelming MySQL
		time.Sleep(500 * time.Millisecond)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("timeout exceeded")
	}
	return nil, fmt.Errorf("failed to connect within timeout: %w", lastErr)
}

func (m *Mysql) GetDB() *sql.DB {
	return m.db
}

func (m *Mysql) GetContainer() *harness.Container {
	return m.container
}

func (m *Mysql) Cleanup() error {
	if m.db != nil {
		m.db.Close()
	}
	return m.container.Cleanup()
}
