// Package api is the HTTP surface: a sync transcribe path (fast, for interactive
// dictation) and an async queued job path (for many devices / batch).
package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/selivanov-tech/go-stt/internal/config"
	"github.com/selivanov-tech/go-stt/internal/engine"
	"github.com/selivanov-tech/go-stt/internal/job"
	"github.com/selivanov-tech/go-stt/internal/queue"
	"github.com/selivanov-tech/go-stt/internal/store"
)

type Server struct {
	cfg   config.Config
	queue queue.Queue
	store store.Store
}

func NewServer(cfg config.Config, q queue.Queue, st store.Store) *Server {
	return &Server{cfg: cfg, queue: q, store: st}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("POST /v1/transcribe", s.handleSync)
	mux.HandleFunc("POST /v1/jobs", s.handleSubmit)
	mux.HandleFunc("GET /v1/jobs/{id}", s.handleJob)
	return mux
}

type transcribeRequest struct {
	AudioURL    string `json:"audio_url,omitempty"`
	AudioPath   string `json:"audio_path,omitempty"`
	AudioBase64 string `json:"audio_base64,omitempty"`
	Language    string `json:"language,omitempty"`
	Engine      string `json:"engine,omitempty"`
	Model       string `json:"model,omitempty"`
}

func (r transcribeRequest) audio() engine.AudioRef {
	return engine.AudioRef{URL: r.AudioURL, Path: r.AudioPath, Base64: r.AudioBase64}
}

func (r transcribeRequest) opts() engine.Options {
	return engine.Options{Language: r.Language, Model: r.Model}
}

func (s *Server) engineFor(spec string) engine.Engine {
	if spec == "" {
		spec = s.cfg.DefaultEngine
	}
	return engine.Build(spec, s.cfg.WorkerURL)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "ok",
		"profile":        s.cfg.Profile,
		"default_engine": s.cfg.DefaultEngine,
		"worker_url":     s.cfg.WorkerURL,
	})
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	var req transcribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	eng := s.engineFor(req.Engine)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	res, err := eng.Transcribe(ctx, req.audio(), req.opts())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	var req transcribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	eng := s.engineFor(req.Engine)
	id := newID()
	engName := eng.Name()
	s.store.Put(job.Job{ID: id, Status: job.StatusQueued, Engine: engName})

	audio, opts := req.audio(), req.opts()
	err := s.queue.Submit(func(ctx context.Context) {
		s.store.Put(job.Job{ID: id, Status: job.StatusRunning, Engine: engName})

		res, err := eng.Transcribe(ctx, audio, opts)
		if err != nil {
			s.store.Put(job.Job{ID: id, Status: job.StatusFailed, Engine: engName, Error: err.Error()})
			return
		}
		s.store.Put(job.Job{ID: id, Status: job.StatusDone, Engine: res.Engine, Text: res.Text, LatencyMS: res.LatencyMS})
	})
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "queue unavailable: "+err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"job_id": id, "status": job.StatusQueued})
}

func (s *Server) handleJob(w http.ResponseWriter, r *http.Request) {
	j, ok := s.store.Get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	writeJSON(w, http.StatusOK, j)
}

func newID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
