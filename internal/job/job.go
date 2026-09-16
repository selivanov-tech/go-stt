package job

import "github.com/selivanov-tech/go-stt/internal/engine"

type Status string

const (
	StatusQueued  Status = "queued"
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusFailed  Status = "failed"
)

type Job struct {
	ID               string           `json:"job_id"`
	Status           Status           `json:"status"`
	Engine           string           `json:"engine,omitempty"`
	Model            string           `json:"model,omitempty"`
	Text             string           `json:"text,omitempty"`
	Error            string           `json:"error,omitempty"`
	LatencyMS        int64            `json:"latency_ms,omitempty"`
	DetectedLanguage string           `json:"detected_language,omitempty"`
	AudioDurationS   float64          `json:"audio_duration_s,omitempty"`
	Segments         []engine.Segment `json:"segments,omitempty"`
}
