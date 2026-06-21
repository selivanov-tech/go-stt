package store

import (
	"testing"

	"github.com/selivanov-tech/go-stt/internal/job"
)

func TestBuild(t *testing.T) {
	for _, kind := range []string{"memory", ""} {
		if s, err := Build(kind); err != nil || s == nil {
			t.Fatalf("Build(%q) = %v, %v; want a store, nil", kind, s, err)
		}
	}
	for _, kind := range []string{"sqlite", "postgres", "nope"} {
		if _, err := Build(kind); err == nil {
			t.Fatalf("Build(%q) = nil error; want an error", kind)
		}
	}
}

func TestMemoryEviction(t *testing.T) {
	m := NewMemoryWithCap(2)
	m.Put(job.Job{ID: "a"})
	m.Put(job.Job{ID: "b"})
	m.Put(job.Job{ID: "c"}) // over cap → evicts oldest ("a")

	if _, ok := m.Get("a"); ok {
		t.Fatal("oldest job 'a' should have been evicted")
	}
	for _, id := range []string{"b", "c"} {
		if _, ok := m.Get(id); !ok {
			t.Fatalf("job %q should still be present", id)
		}
	}

	m.Put(job.Job{ID: "b", Status: job.StatusDone}) // update, not a new entry
	if _, ok := m.Get("c"); !ok {
		t.Fatal("updating 'b' should not evict 'c'")
	}
}
