package seekdb

import (
	"database/sql"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// EmbeddedConfig holds configuration for starting seekdb in embedded mode.
type EmbeddedConfig struct {
	BinaryPath  string            // Path to the seekdb binary (required)
	BaseDir     string            // Data directory for seekdb
	Port        int               // Port to listen on (0 = auto-assign)
	ExtraParams map[string]string // Additional command-line parameters passed to the seekdb binary
	LogLevel    string            // Log level for the seekdb subprocess (e.g., INFO, DEBUG)
}

// EmbeddedProcess manages a seekdb instance running as a subprocess.
// Starts the seekdb binary as a child process and connects via MySQL protocol.
type EmbeddedProcess struct {
	mu          sync.Mutex
	baseDir     string
	port        int
	binary      string
	extraParams map[string]string
	logLevel    string
	cmd         *exec.Cmd
	stderr      io.ReadCloser
	started     bool
	stopped     bool
	done        chan struct{}
	exitErr     error
}

// NewEmbeddedProcess creates a new EmbeddedProcess with the given configuration.
func NewEmbeddedProcess(cfg EmbeddedConfig) (*EmbeddedProcess, error) {
	if cfg.BaseDir == "" {
		return nil, fmt.Errorf("BaseDir is required for embedded mode")
	}

	binary := cfg.BinaryPath
	if binary == "" {
		binary = os.Getenv(EnvBinaryPath)
	}
	if binary == "" {
		var err error
		binary, err = exec.LookPath("seekdb")
		if err != nil {
			return nil, ErrBinaryNotFound
		}
	}

	p := cfg.Port
	if p == 0 {
		var err error
		p, err = findFreePort()
		if err != nil {
			return nil, fmt.Errorf("failed to find free port: %w", err)
		}
	}

	return &EmbeddedProcess{
		baseDir:     cfg.BaseDir,
		port:        p,
		binary:      binary,
		extraParams: cfg.ExtraParams,
		logLevel:    cfg.LogLevel,
	}, nil
}

// Start launches the seekdb subprocess and waits for it to accept connections.
func (ep *EmbeddedProcess) Start(timeout time.Duration) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if ep.started {
		return nil
	}

	args := ep.buildArgs()

	ep.cmd = exec.Command(ep.binary, args...)

	stderrPipe, err := ep.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	ep.stderr = stderrPipe

	ep.done = make(chan struct{})
	go func() {
		defer close(ep.done)
		io.Copy(os.Stderr, stderrPipe)
	}()

	if err := ep.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start seekdb process: %w", err)
	}

	if err := ep.waitForReady(timeout); err != nil {
		ep.cmd.Process.Kill()
		ep.exitErr = err
		return fmt.Errorf("seekdb failed to become ready: %w", err)
	}

	ep.started = true
	return nil
}

// Stop terminates the seekdb subprocess.
func (ep *EmbeddedProcess) Stop() error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if ep.stopped || !ep.started {
		return nil
	}

	if ep.cmd != nil && ep.cmd.Process != nil {
		ep.cmd.Process.Signal(syscall.SIGTERM)

		select {
		case <-ep.done:
		case <-time.After(5 * time.Second):
			ep.cmd.Process.Kill()
			<-ep.done
		}
	}

	ep.stopped = true
	ep.started = false
	return ep.exitErr
}

// Port returns the port the embedded seekdb is listening on.
func (ep *EmbeddedProcess) Port() int {
	return ep.port
}

// BaseDir returns the base directory.
func (ep *EmbeddedProcess) BaseDir() string {
	return ep.baseDir
}

// Binary returns the resolved binary path.
func (ep *EmbeddedProcess) Binary() string {
	return ep.binary
}

// IsRunning returns true if the instance is running.
func (ep *EmbeddedProcess) IsRunning() bool {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	return ep.started && !ep.stopped
}

// Connect opens a MySQL connection to the embedded seekdb instance.
func (ep *EmbeddedProcess) Connect(database string, poolConfig ConnectionPoolConfig) (*sql.DB, error) {
	if !ep.IsRunning() {
		return nil, ErrNotConnected
	}

	dsn := fmt.Sprintf("root:@tcp(127.0.0.1:%d)/%s", ep.port, database)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	if poolConfig.MaxOpenConns == 0 {
		poolConfig = DefaultConnectionPoolConfig()
	}
	db.SetMaxOpenConns(poolConfig.MaxOpenConns)
	db.SetMaxIdleConns(poolConfig.MaxIdleConns)
	if poolConfig.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(poolConfig.ConnMaxLifetime)
	}
	if poolConfig.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(poolConfig.ConnMaxIdleTime)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping embedded seekdb: %w", err)
	}

	return db, nil
}

// ConnectAdmin opens a MySQL connection without a database (for admin operations).
func (ep *EmbeddedProcess) ConnectAdmin(poolConfig ConnectionPoolConfig) (*sql.DB, error) {
	if !ep.IsRunning() {
		return nil, ErrNotConnected
	}

	dsn := fmt.Sprintf("root:@tcp(127.0.0.1:%d)/", ep.port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	if poolConfig.MaxOpenConns == 0 {
		poolConfig = DefaultConnectionPoolConfig()
	}
	db.SetMaxOpenConns(poolConfig.MaxOpenConns)
	db.SetMaxIdleConns(poolConfig.MaxIdleConns)
	if poolConfig.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(poolConfig.ConnMaxLifetime)
	}
	if poolConfig.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(poolConfig.ConnMaxIdleTime)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping embedded seekdb: %w", err)
	}

	return db, nil
}

// buildArgs constructs command-line arguments for the seekdb subprocess.
func (ep *EmbeddedProcess) buildArgs() []string {
	args := []string{
		"--datadir=" + ep.baseDir,
		"--port=" + fmt.Sprintf("%d", ep.port),
	}

	if ep.logLevel != "" {
		args = append(args, "--log-level="+ep.logLevel)
	}

	for key, value := range ep.extraParams {
		args = append(args, "--"+key+"="+value)
	}

	return args
}

// waitForReady polls until seekdb responds on the network port or timeout.
func (ep *EmbeddedProcess) waitForReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timed out waiting for seekdb to be ready on port %d", ep.port)
			}

			// Check if process already exited
			if ep.cmd != nil && ep.cmd.ProcessState != nil && ep.cmd.ProcessState.Exited() {
				return fmt.Errorf("seekdb process exited unexpectedly (exit code: %d)", ep.cmd.ProcessState.ExitCode())
			}

			// Check TCP connectivity
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", ep.port), 500*time.Millisecond)
			if err == nil {
				conn.Close()
				return nil
			}
		}
	}
}

// findFreePort finds an available TCP port.
func findFreePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port, nil
}
