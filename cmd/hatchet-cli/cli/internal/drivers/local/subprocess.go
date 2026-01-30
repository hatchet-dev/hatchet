package local

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
)

// SubprocessManager manages API and Engine subprocesses
type SubprocessManager struct {
	apiCmd    *exec.Cmd
	engineCmd *exec.Cmd
	mu        sync.Mutex
	started   bool
}

// NewSubprocessManager creates a new subprocess manager
func NewSubprocessManager() *SubprocessManager {
	return &SubprocessManager{}
}

// StartAPI starts the API server as a subprocess
func (sm *SubprocessManager) StartAPI(ctx context.Context, binaryPath string, env []string, configDir string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.apiCmd != nil {
		return fmt.Errorf("API is already running")
	}

	fmt.Println(styles.InfoMessage("Starting API subprocess..."))

	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Env = append(os.Environ(), env...)
	cmd.Dir = configDir

	// Pipe stdout and stderr to the current process
	cmd.Stdout = newPrefixWriter(os.Stdout, "[api] ")
	cmd.Stderr = newPrefixWriter(os.Stderr, "[api] ")

	// Set platform-specific process attributes
	setPlatformAttributes(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start API: %w", err)
	}

	sm.apiCmd = cmd
	fmt.Println(styles.SuccessMessage(fmt.Sprintf("API started (PID: %d)", cmd.Process.Pid)))

	return nil
}

// StartEngine starts the Engine as a subprocess
func (sm *SubprocessManager) StartEngine(ctx context.Context, binaryPath string, env []string, configDir string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.engineCmd != nil {
		return fmt.Errorf("Engine is already running")
	}

	fmt.Println(styles.InfoMessage("Starting Engine subprocess..."))

	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Env = append(os.Environ(), env...)
	cmd.Dir = configDir

	// Pipe stdout and stderr to the current process
	cmd.Stdout = newPrefixWriter(os.Stdout, "[engine] ")
	cmd.Stderr = newPrefixWriter(os.Stderr, "[engine] ")

	// Set platform-specific process attributes
	setPlatformAttributes(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Engine: %w", err)
	}

	sm.engineCmd = cmd
	sm.started = true
	fmt.Println(styles.SuccessMessage(fmt.Sprintf("Engine started (PID: %d)", cmd.Process.Pid)))

	return nil
}

// StopAll gracefully stops all running subprocesses
func (sm *SubprocessManager) StopAll() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var errs []error

	if sm.apiCmd != nil && sm.apiCmd.Process != nil {
		fmt.Println(styles.InfoMessage("Stopping API subprocess..."))
		if err := stopProcess(sm.apiCmd); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop API: %w", err))
		}
		sm.apiCmd = nil
	}

	if sm.engineCmd != nil && sm.engineCmd.Process != nil {
		fmt.Println(styles.InfoMessage("Stopping Engine subprocess..."))
		if err := stopProcess(sm.engineCmd); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop Engine: %w", err))
		}
		sm.engineCmd = nil
	}

	sm.started = false

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

// WaitForHealth waits for the API to become healthy
func (sm *SubprocessManager) WaitForHealth(ctx context.Context, apiPort int) error {
	healthURL := fmt.Sprintf("http://localhost:%d/api/ready", apiPort)
	healthCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	client := &http.Client{Timeout: 2 * time.Second}

	fmt.Println(styles.InfoMessage("Waiting for API to become healthy..."))

	for {
		select {
		case <-healthCtx.Done():
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("timeout waiting for API to become healthy")
		case <-ticker.C:
			// Check if processes are still running
			sm.mu.Lock()
			apiRunning := sm.apiCmd != nil && sm.apiCmd.ProcessState == nil
			engineRunning := sm.engineCmd != nil && sm.engineCmd.ProcessState == nil
			sm.mu.Unlock()

			if !apiRunning || !engineRunning {
				return fmt.Errorf("subprocess exited unexpectedly")
			}

			req, err := http.NewRequestWithContext(healthCtx, "GET", healthURL, nil)
			if err != nil {
				continue
			}

			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					fmt.Println(styles.SuccessMessage("API is healthy"))
					return nil
				}
			}
		}
	}
}

// Wait waits for all subprocesses to exit
func (sm *SubprocessManager) Wait() error {
	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	sm.mu.Lock()
	apiCmd := sm.apiCmd
	engineCmd := sm.engineCmd
	sm.mu.Unlock()

	if apiCmd != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := apiCmd.Wait(); err != nil {
				errCh <- fmt.Errorf("API exited: %w", err)
			}
		}()
	}

	if engineCmd != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := engineCmd.Wait(); err != nil {
				errCh <- fmt.Errorf("Engine exited: %w", err)
			}
		}()
	}

	// Wait for first error or all processes to exit
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case err := <-errCh:
		return err
	case <-done:
		return nil
	}
}

// IsRunning returns whether subprocesses are currently running
func (sm *SubprocessManager) IsRunning() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.started
}

// prefixWriter wraps an io.Writer and adds a prefix to each line
type prefixWriter struct {
	w      io.Writer
	prefix string
	buf    []byte
}

func newPrefixWriter(w io.Writer, prefix string) *prefixWriter {
	return &prefixWriter{
		w:      w,
		prefix: prefix,
		buf:    make([]byte, 0, 1024),
	}
}

func (pw *prefixWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	pw.buf = append(pw.buf, p...)

	for {
		idx := -1
		for i, b := range pw.buf {
			if b == '\n' {
				idx = i
				break
			}
		}

		if idx == -1 {
			break
		}

		line := pw.buf[:idx+1]
		pw.buf = pw.buf[idx+1:]

		if _, err := fmt.Fprintf(pw.w, "%s%s", pw.prefix, line); err != nil {
			return n, err
		}
	}

	return n, nil
}
