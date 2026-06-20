// Package store keeps job state. The in-memory impl is fine for a single node;
// swap for SQLite/Postgres when you need durability across restarts.
package store

import (
	"fmt"

	"github.com/selivanov-tech/go-stt/internal/job"
)

type Store interface {
	Put(j job.Job)
	Get(id string) (job.Job, bool)
}

// Build selects a store backend; sqlite/postgres are stubs until their adapters land.
func Build(kind string) (Store, error) {
	switch kind {
	case "", "memory":
		return NewMemory(), nil
	case "sqlite", "postgres":
		return nil, fmt.Errorf("store %q is not implemented yet", kind)
	default:
		return nil, fmt.Errorf("unknown store %q (use memory|sqlite|postgres)", kind)
	}
}
