package engine

import (
	"context"
	"fmt"
)

// CloudEngine is a placeholder for direct Go calls to a cloud STT API (Deepgram,
// AssemblyAI, ...), used in the production profile where the Python worker is absent.
type CloudEngine struct {
	provider string
}

func NewCloudEngine(provider string) *CloudEngine { return &CloudEngine{provider: provider} }

func (e *CloudEngine) Name() string { return e.provider }

func (e *CloudEngine) Transcribe(ctx context.Context, audio AudioRef, opts Options) (Result, error) {
	return Result{}, fmt.Errorf("cloud engine %q not implemented yet: add its Go client + API key", e.provider)
}
