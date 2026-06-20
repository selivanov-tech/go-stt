package store

import "testing"

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
