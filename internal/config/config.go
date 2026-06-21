package config

import (
	"fmt"
	"os"
	"strconv"
)

type Profile string

const (
	ProfileLocal      Profile = "local"
	ProfileProduction Profile = "production"
)

type Config struct {
	Addr            string  // HTTP listen address
	Profile         Profile // local | production
	DefaultEngine   string  // engine spec used when a request omits one
	Store           string  // job store: memory | sqlite | postgres
	WorkerURL       string  // Python model-worker base URL (local profile)
	WorkerCmd       string  // optional command to autostart the worker
	WorkerDir       string  // working dir for the worker command
	WorkerAutostart bool
	QueueWorkers    int // worker-pool size
	QueueBuffer     int // queued-jobs buffer before Submit blocks
}

func Load() Config {
	return Config{
		Addr:            getenv("STT_ADDR", ":8080"),
		Profile:         Profile(getenv("STT_PROFILE", "local")),
		DefaultEngine:   getenv("STT_DEFAULT_ENGINE", "python:mock"),
		Store:           getenv("STT_STORE", "memory"),
		WorkerURL:       getenv("STT_WORKER_URL", "http://127.0.0.1:8090"),
		WorkerCmd:       getenv("STT_WORKER_CMD", ""),
		WorkerDir:       getenv("STT_WORKER_DIR", "../py-stt"),
		WorkerAutostart: getenv("STT_WORKER_AUTOSTART", "false") == "true",
		QueueWorkers:    atoiDefault(getenv("STT_QUEUE_WORKERS", "2"), 2),
		QueueBuffer:     atoiDefault(getenv("STT_QUEUE_BUFFER", "256"), 256),
	}
}

// Validate reports config that would be unsafe to run. Call it before start.
func (c Config) Validate() error {
	switch c.Profile {
	case ProfileLocal, ProfileProduction:
	default:
		return fmt.Errorf("unknown profile %q (use local|production)", c.Profile)
	}
	durable := c.Store == "sqlite" || c.Store == "postgres"
	if c.requiresDurableStore() && !durable {
		return fmt.Errorf(
			"profile %q needs a durable job store: set STT_STORE=sqlite|postgres "+
				"(the in-memory store loses jobs on restart and grows without bound)",
			c.Profile,
		)
	}
	return nil
}

func (c Config) requiresDurableStore() bool {
	return c.Profile != ProfileLocal
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func atoiDefault(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}
