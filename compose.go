package dockerharness

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultComposeWaitTimeout = 60 * time.Second

type ComposeOptions struct {
	Name          string
	Files         []string
	WorkDir       string
	Env           map[string]string
	Profiles      []string
	Services      []string
	Wait          bool
	NoWait        bool
	WaitTimeout   time.Duration
	Build         bool
	Pull          string
	KeepVolumes   bool
	RemoveOrphans bool
	KeepOrphans   bool
	Stdout        io.Writer
	Stderr        io.Writer
}

type Compose struct {
	name          string
	files         []string
	workDir       string
	env           map[string]string
	profiles      []string
	services      []string
	wait          bool
	waitTimeout   time.Duration
	build         bool
	pull          string
	keepVolumes   bool
	removeOrphans bool
	stdout        io.Writer
	stderr        io.Writer
	command       []string

	lock sync.Mutex
}

type ComposeContainer struct {
	ID       string `json:"ID"`
	Name     string `json:"Name"`
	Command  string `json:"Command"`
	Project  string `json:"Project"`
	Service  string `json:"Service"`
	State    string `json:"State"`
	Health   string `json:"Health"`
	ExitCode int    `json:"ExitCode"`
}

/*
NewCompose will create a new docker compose harness from a list of compose
files. If a name is not provided, a unique project name will be generated.
*/
func NewCompose(name string, files []string) (*Compose, error) {
	return NewComposeWithOptions(ComposeOptions{
		Name:  name,
		Files: files,
	})
}

/*
NewHarnessFromFiles will create a new harness from docker compose files.
*/
func NewHarnessFromFiles(name string, files []string) (Harness, error) {
	return NewCompose(name, files)
}

/*
NewComposeWithOptions will create a new docker compose harness with
additional docker compose options.
*/
func NewComposeWithOptions(options ComposeOptions) (*Compose, error) {
	if len(options.Files) == 0 {
		return nil, errors.New("at least one compose file is required")
	}
	for _, file := range options.Files {
		if file == "" {
			return nil, errors.New("compose file path cannot be blank")
		}
	}

	command, err := detectComposeCommand()
	if err != nil {
		return nil, err
	}

	name := options.Name
	if name == "" {
		name = generateComposeName()
	}

	workDir := options.WorkDir
	if workDir == "" {
		workDir = "."
	}

	wait := options.Wait
	if !options.NoWait {
		wait = true
	}

	waitTimeout := options.WaitTimeout
	if waitTimeout == 0 {
		waitTimeout = defaultComposeWaitTimeout
	}

	removeOrphans := options.RemoveOrphans
	if !options.KeepOrphans {
		removeOrphans = true
	}

	return &Compose{
		name:          name,
		files:         options.Files,
		workDir:       workDir,
		env:           options.Env,
		profiles:      options.Profiles,
		services:      options.Services,
		wait:          wait,
		waitTimeout:   waitTimeout,
		build:         options.Build,
		pull:          options.Pull,
		keepVolumes:   options.KeepVolumes,
		removeOrphans: removeOrphans,
		stdout:        options.Stdout,
		stderr:        options.Stderr,
		command:       command,
	}, nil
}

/*
Start will create and start the compose project. By default, docker compose
will wait until services are running or healthy before returning.
*/
func (c *Compose) Start() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	args := []string{"up", "--detach"}
	if c.wait {
		args = append(args, "--wait")
		if c.waitTimeout > 0 {
			args = append(args, "--wait-timeout", strconv.Itoa(int(c.waitTimeout.Seconds())))
		}
	}
	if c.build {
		args = append(args, "--build")
	}
	if c.pull != "" {
		args = append(args, "--pull", c.pull)
	}
	if c.removeOrphans {
		args = append(args, "--remove-orphans")
	}
	args = append(args, c.services...)

	return c.run(args...)
}

/*
Stop will send a stop signal to all running containers in the compose
project. If wait is greater than zero, docker compose will wait up to
that many seconds for containers to stop.
*/
func (c *Compose) Stop(wait int) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	args := []string{"stop"}
	if wait > 0 {
		args = append(args, "--timeout", strconv.Itoa(wait))
	}
	args = append(args, c.services...)

	return c.run(args...)
}

/*
Cleanup will stop and remove all containers and networks for the compose
project. By default, volumes created by the compose file are also removed.
*/
func (c *Compose) Cleanup() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	args := []string{"down"}
	if c.removeOrphans {
		args = append(args, "--remove-orphans")
	}
	if !c.keepVolumes {
		args = append(args, "--volumes")
	}

	return c.run(args...)
}

/*
IsRunning will return true if any container in the compose project is
running, false otherwise.
*/
func (c *Compose) IsRunning() (bool, error) {
	containers, err := c.GetContainers()
	if err != nil {
		return false, err
	}

	for _, container := range containers {
		if strings.ToLower(container.State) == "running" {
			return true, nil
		}
	}

	return false, nil
}

/*
GetServices will return the services created by the compose project.
*/
func (c *Compose) GetServices() ([]string, error) {
	out, err := c.output("ps", "--services")
	if err != nil {
		return nil, err
	}

	services := []string{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		service := strings.TrimSpace(line)
		if service != "" {
			services = append(services, service)
		}
	}

	return services, nil
}

/*
GetContainers will return containers created by the compose project.
*/
func (c *Compose) GetContainers() ([]ComposeContainer, error) {
	out, err := c.output("ps", "--all", "--format", "json")
	if err != nil {
		return nil, err
	}
	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		return []ComposeContainer{}, nil
	}

	containers := []ComposeContainer{}
	if err := json.Unmarshal(out, &containers); err == nil {
		return containers, nil
	}

	decoder := json.NewDecoder(bytes.NewReader(out))
	for {
		container := ComposeContainer{}
		if err := decoder.Decode(&container); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		containers = append(containers, container)
	}

	return containers, nil
}

/*
GetPort will return the host address for a service's private port.
*/
func (c *Compose) GetPort(service string, privatePort int, protocol string) (string, error) {
	if service == "" {
		return "", errors.New("service is required")
	}
	if privatePort <= 0 {
		return "", errors.New("private port must be greater than zero")
	}
	if protocol == "" {
		protocol = "tcp"
	}

	out, err := c.output("port", "--protocol", protocol, service, strconv.Itoa(privatePort))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

/*
GetLogs will return logs for the compose project. If services are provided,
only those service logs will be returned.
*/
func (c *Compose) GetLogs(services ...string) (string, error) {
	args := []string{"logs", "--no-color"}
	args = append(args, services...)

	out, err := c.output(args...)
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func (c *Compose) GetName() string {
	return c.name
}

func (c *Compose) GetFiles() []string {
	files := make([]string, len(c.files))
	copy(files, c.files)
	return files
}

func (c *Compose) run(args ...string) error {
	cmd, stderr := c.commandContext(false, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (c *Compose) output(args ...string) ([]byte, error) {
	cmd, stderr := c.commandContext(true, args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker compose %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func (c *Compose) commandContext(captureOutput bool, args ...string) (*exec.Cmd, *bytes.Buffer) {
	baseArgs := []string{}
	for _, file := range c.files {
		baseArgs = append(baseArgs, "--file", file)
	}
	baseArgs = append(baseArgs, "--project-name", c.name)
	for _, profile := range c.profiles {
		baseArgs = append(baseArgs, "--profile", profile)
	}
	baseArgs = append(baseArgs, args...)

	cmd := exec.CommandContext(context.Background(), c.command[0], append(c.command[1:], baseArgs...)...)
	cmd.Dir = c.workDir
	cmd.Env = os.Environ()
	for k, v := range c.env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	if c.stdout != nil && !captureOutput {
		cmd.Stdout = c.stdout
	}

	stderr := &bytes.Buffer{}
	if c.stderr != nil {
		cmd.Stderr = io.MultiWriter(c.stderr, stderr)
	} else {
		cmd.Stderr = stderr
	}

	return cmd, stderr
}

func detectComposeCommand() ([]string, error) {
	if err := exec.Command("docker", "compose", "version").Run(); err == nil {
		return []string{"docker", "compose"}, nil
	}
	if err := exec.Command("docker-compose", "version").Run(); err == nil {
		return []string{"docker-compose"}, nil
	}
	return nil, errors.New("docker compose is not available")
}

func generateComposeName() string {
	return fmt.Sprintf("docker-harness-%d-%d", time.Now().UnixNano(), rand.IntN(1000000))
}
