package store

import (
	"sync"

	"github.com/selivanov-tech/go-stt/internal/job"
)

type Memory struct {
	mu   sync.RWMutex
	jobs map[string]job.Job
}

func NewMemory() *Memory {
	return &Memory{jobs: make(map[string]job.Job)}
}

func (m *Memory) Put(j job.Job) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[j.ID] = j
}

func (m *Memory) Get(id string) (job.Job, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	j, ok := m.jobs[id]
	return j, ok
}
