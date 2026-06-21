package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/selivanov-tech/go-stt/internal/api"
	"github.com/selivanov-tech/go-stt/internal/config"
	"github.com/selivanov-tech/go-stt/internal/queue"
	"github.com/selivanov-tech/go-stt/internal/store"
)

// fakeWorker stands in for the py-stt model worker.
func fakeWorker() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"text":              "hello world",
			"engine":            "mock",
			"model":             "echo",
			"latency_ms":        1,
			"detected_language": "en",
		})
	}))
}

func newTestServer(t *testing.T, workerURL string) *httptest.Server {
	t.Helper()
	cfg := config.Config{
		Profile:       config.ProfileLocal,
		DefaultEngine: "python:mock",
		WorkerURL:     workerURL,
	}
	q := queue.NewInProc(2, 16)
	ts := httptest.NewServer(api.NewServer(cfg, q, store.NewMemory()).Routes())
	t.Cleanup(func() {
		ts.Close()
		_ = q.Shutdown(context.Background())
	})
	return ts
}

func TestHealthz(t *testing.T) {
	worker := fakeWorker()
	defer worker.Close()
	ts := newTestServer(t, worker.URL)

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz = %d, want 200", resp.StatusCode)
	}
}

func TestSyncTranscribe(t *testing.T) {
	worker := fakeWorker()
	defer worker.Close()
	ts := newTestServer(t, worker.URL)

	resp, err := http.Post(ts.URL+"/v1/transcribe", "application/json",
		bytes.NewReader([]byte(`{"audio_path":"clip.wav","language":"en"}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("transcribe = %d (%s), want 200", resp.StatusCode, b)
	}
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out["text"] != "hello world" {
		t.Fatalf("text = %v, want 'hello world'", out["text"])
	}
}

func TestTranscribeRequiresAudio(t *testing.T) {
	worker := fakeWorker()
	defer worker.Close()
	ts := newTestServer(t, worker.URL)

	resp, err := http.Post(ts.URL+"/v1/transcribe", "application/json",
		bytes.NewReader([]byte(`{"language":"en"}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("no-audio transcribe = %d, want 400", resp.StatusCode)
	}
}

func TestJobLifecycle(t *testing.T) {
	worker := fakeWorker()
	defer worker.Close()
	ts := newTestServer(t, worker.URL)

	resp, err := http.Post(ts.URL+"/v1/jobs", "application/json",
		bytes.NewReader([]byte(`{"audio_path":"clip.wav"}`)))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("submit = %d, want 202", resp.StatusCode)
	}
	var submitted struct {
		JobID string `json:"job_id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&submitted)
	_ = resp.Body.Close()
	if submitted.JobID == "" {
		t.Fatal("no job_id returned")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		r, err := http.Get(ts.URL + "/v1/jobs/" + submitted.JobID)
		if err != nil {
			t.Fatal(err)
		}
		var jb struct {
			Status string `json:"status"`
			Text   string `json:"text"`
		}
		_ = json.NewDecoder(r.Body).Decode(&jb)
		_ = r.Body.Close()
		switch jb.Status {
		case "done":
			if jb.Text != "hello world" {
				t.Fatalf("job text = %q, want 'hello world'", jb.Text)
			}
			return
		case "failed":
			t.Fatal("job failed unexpectedly")
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("job did not reach 'done' within 2s")
}

func TestUnknownJob404(t *testing.T) {
	worker := fakeWorker()
	defer worker.Close()
	ts := newTestServer(t, worker.URL)

	resp, err := http.Get(ts.URL + "/v1/jobs/deadbeef")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown job = %d, want 404", resp.StatusCode)
	}
}
