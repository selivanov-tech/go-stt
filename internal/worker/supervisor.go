// Package worker optionally launches and supervises the Python model-worker, so
// `./stt-service` can boot the whole local stack with one command.
package worker

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
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

func (s *Supervisor) Start() error {
	if strings.TrimSpace(s.cmdline) == "" {
		return nil
	}
	parts := strings.Fields(s.cmdline)
	c := exec.Command(parts[0], parts[1:]...)
	c.Dir = s.dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Start(); err != nil {
		return fmt.Errorf("start worker: %w", err)
	}
	s.cmd = c
	return nil
}

func (s *Supervisor) Stop() {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
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
