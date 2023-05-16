package postgres

import (
	"database/sql"
	"fmt"
	"time"

	harness "github.com/hlfshell/docker-harness"

	_ "github.com/lib/pq"
)

type Postgres struct {
	container *harness.Container
	db        *sql.DB
	username  string
	password  string
	database  string
	port      string
}

func NewPostgres(name string, tag string, username string, password string, database string) (*Postgres, error) {
	container, err := harness.NewContainer(
		name,
		"postgres",
		tag,
		map[string]string{
			"5432": "",
		},
		map[string]string{
			"POSTGRES_USER":     username,
			"POSTGRES_PASSWORD": password,
			"POSTGRES_DB":       database,
		},
	)
	if err != nil {
		return nil, err
	}
	return &Postgres{
		container: container,
		username:  username,
		password:  password,
		database:  database,
	}, nil
}

func (p *Postgres) Create() (*sql.DB, error) {
	err := p.container.Start()
	if err != nil {
		return nil, err
	}

	// Grab the assigned port
	ports := p.container.GetPorts()
	p.port = ports["5432"]

	// Connect to the database, but have a built in retry
	// / timeout due to startup time of the container
	start := time.Now()
	timeout := 10 * time.Second
	running := false
	for !running && time.Since(start) < timeout {
		running, err = p.container.IsRunning()
		if err != nil {
			p.container.Cleanup()
			return nil, err
		}
	}
	if !running {
		p.container.Cleanup()
		return nil, fmt.Errorf("container failed to start within timeout")
	}

	// Now that the container is running, attempt to create
	// a db connection
	var db *sql.DB
	for time.Since(start) < timeout {
		db, err = p.Connect()
		if err == nil {
			break
		}
	}
	if err != nil {
		p.container.Cleanup()
	}
	return db, err
}

func (p *Postgres) Connect() (*sql.DB, error) {
	connectionString := fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		p.username,
		p.password,
		"0.0.0.0",
		p.port,
		p.database,
	)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}
	p.db = db
	return db, nil
}

func (p *Postgres) GetDB() *sql.DB {
	return p.db
}

func (p *Postgres) GetContainer() *harness.Container {
	return p.container
}

func (p *Postgres) Cleanup() error {
	return p.container.Cleanup()
}
