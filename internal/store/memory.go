package store

import (
	"sync"

	"github.com/selivanov-tech/go-stt/internal/job"
)

const defaultMaxJobs = 10000

type Memory struct {
	mu    sync.RWMutex
	jobs  map[string]job.Job
	order []string
	max   int
}

func NewMemory() *Memory { return NewMemoryWithCap(defaultMaxJobs) }

// NewMemoryWithCap bounds the store to max jobs, evicting the oldest first.
// max <= 0 means unbounded.
func NewMemoryWithCap(max int) *Memory {
	return &Memory{jobs: make(map[string]job.Job), max: max}
}

func (m *Memory) Put(j job.Job) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.jobs[j.ID]; !exists {
		m.order = append(m.order, j.ID)
		for m.max > 0 && len(m.order) > m.max {
			oldest := m.order[0]
			m.order = m.order[1:]
			delete(m.jobs, oldest)
		}
	}
	m.jobs[j.ID] = j
}

func (m *Memory) Get(id string) (job.Job, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	j, ok := m.jobs[id]
	return j, ok
}
