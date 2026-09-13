package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PythonEngine calls the model-worker over HTTP; pyEngine picks the model
// (mock, faster_whisper, mlx_whisper, ...).
type PythonEngine struct {
	pyEngine string
	name     string
	baseURL  string
	model    string
	client   *http.Client
}

func NewPythonEngine(pyEngine, baseURL, model string, timeout time.Duration) *PythonEngine {
	return &PythonEngine{
		pyEngine: pyEngine,
		name:     "python:" + pyEngine,
		baseURL:  baseURL,
		model:    model,
		client:   &http.Client{Timeout: timeout},
	}
}

func (e *PythonEngine) Name() string { return e.name }

type pyRequest struct {
	AudioURL    string `json:"audio_url,omitempty"`
	AudioPath   string `json:"audio_path,omitempty"`
	AudioBase64 string `json:"audio_base64,omitempty"`
	Language    string `json:"language,omitempty"`
	Model       string `json:"model,omitempty"`
	Engine      string `json:"engine"`
}

func (e *PythonEngine) Transcribe(ctx context.Context, audio AudioRef, opts Options) (Result, error) {
	model := opts.Model
	if model == "" {
		model = e.model
	}
	body, _ := json.Marshal(pyRequest{
		AudioURL:    audio.URL,
		AudioPath:   audio.Path,
		AudioBase64: audio.Base64,
		Language:    opts.Language,
		Model:       model,
		Engine:      e.pyEngine,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/transcribe", bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("worker call failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("worker returned %d: %s", resp.StatusCode, string(data))
	}

	var out Result
	if err := json.Unmarshal(data, &out); err != nil {
		return Result{}, fmt.Errorf("decode worker response: %w", err)
	}
	if out.Engine == "" {
		out.Engine = e.name
	}
	return out, nil
}
