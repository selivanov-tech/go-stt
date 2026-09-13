package config

import (
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		profile Profile
		store   string
		wantErr bool
	}{
		{"local + memory", ProfileLocal, "memory", false},
		{"local + empty defaults ok", ProfileLocal, "", false},
		{"production + memory refused", ProfileProduction, "memory", true},
		{"production + empty refused", ProfileProduction, "", true},
		{"production + postgres ok", ProfileProduction, "postgres", false},
		{"production + sqlite ok", ProfileProduction, "sqlite", false},
		{"unknown profile refused", Profile("staging"), "postgres", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := Config{Profile: tc.profile, Store: tc.store}
			if err := c.Validate(); (err != nil) != tc.wantErr {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestLoadEngineTimeout(t *testing.T) {
	cases := []struct {
		env  string
		want time.Duration
	}{
		{"45m", 45 * time.Minute},
		{"", DefaultEngineTimeout},
		{"soon", DefaultEngineTimeout},
		{"-1m", DefaultEngineTimeout},
	}
	for _, tc := range cases {
		t.Run(tc.env, func(t *testing.T) {
			t.Setenv("STT_ENGINE_TIMEOUT", tc.env)
			if got := Load().EngineTimeout; got != tc.want {
				t.Fatalf("EngineTimeout = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLoadListensOnLoopbackByDefault(t *testing.T) {
	t.Setenv("STT_ADDR", "")
	if got := Load().Addr; got != "127.0.0.1:8080" {
		t.Fatalf("Addr = %q, want 127.0.0.1:8080", got)
	}
}
