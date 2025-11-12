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

	// Give MySQL additional time to fully initialize before connection attempts
	time.Sleep(2 * time.Second)
	return nil
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

	// Ping the database to ensure it is ready
	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	m.db = db

	return db, nil
}

func (m *Mysql) ConnectWithTimeout(timeout time.Duration) (*sql.DB, error) {
	start := time.Now()
	var db *sql.DB
	var err error

	for time.Since(start) < timeout {
		db, err = m.Connect()
		if err == nil && db != nil {
			break
		}
		// Wait longer between retries to reduce connection attempts
		// and give MySQL more time to become ready
		time.Sleep(500 * time.Millisecond)
	}

	return db, err
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
