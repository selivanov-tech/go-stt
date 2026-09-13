// Package engine is the transport-agnostic STT engine interface and its impls.
// Swapping HTTP for gRPC later is a new Engine impl, not a caller change.
package engine

import "context"

// AudioRef points at audio to transcribe; exactly one field is set.
type AudioRef struct {
	URL    string `json:"audio_url,omitempty"`
	Path   string `json:"audio_path,omitempty"`
	Base64 string `json:"audio_base64,omitempty"`
}

type Options struct {
	Language string `json:"language,omitempty"`
	Model    string `json:"model,omitempty"`
}

// Segment is a timed piece of a transcript: a whisper segment or a parakeet sentence.
type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type Result struct {
	Text             string    `json:"text"`
	Engine           string    `json:"engine"`
	Model            string    `json:"model,omitempty"`
	LatencyMS        int64     `json:"latency_ms"`
	DetectedLanguage string    `json:"detected_language,omitempty"`
	AudioDurationS   float64   `json:"audio_duration_s,omitempty"`
	Segments         []Segment `json:"segments,omitempty"`
}

type Engine interface {
	Name() string
	Transcribe(ctx context.Context, audio AudioRef, opts Options) (Result, error)
}
