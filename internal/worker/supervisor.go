// Package worker can launch and supervise the Python model-worker, so the
// whole local stack boots with one command.
package worker

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type Supervisor struct {
	cmdline string
	dir     string
	cmd     *exec.Cmd
}

func New(cmdline, dir string) *Supervisor {
	return &Supervisor{cmdline: cmdline, dir: dir}
}

// Start runs the worker in its own process group, so Stop can signal the whole
// tree. cmdline is split on spaces — fine for the simple commands we configure.
func (s *Supervisor) Start() error {
	if strings.TrimSpace(s.cmdline) == "" {
		return nil
	}
	parts := strings.Fields(s.cmdline)
	c := exec.Command(parts[0], parts[1:]...)
	c.Dir = s.dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := c.Start(); err != nil {
		return fmt.Errorf("start worker: %w", err)
	}
	s.cmd = c
	return nil
}

// Stop signals the worker's whole process group: SIGTERM, then SIGKILL if it
// does not exit within a few seconds.
func (s *Supervisor) Stop() {
	if s.cmd == nil || s.cmd.Process == nil {
		return
	}
	pgid, err := syscall.Getpgid(s.cmd.Process.Pid)
	if err != nil {
		pgid = s.cmd.Process.Pid
	}
	_ = syscall.Kill(-pgid, syscall.SIGTERM)

	done := make(chan struct{})
	go func() {
		_, _ = s.cmd.Process.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}
}

func WaitHealthy(baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(baseURL + "/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("worker not healthy at %s within %s", baseURL, timeout)
}
