package engine

import (
	"strings"
	"time"
)

// Build returns an Engine for a spec:
//   - "python:<key>" -> the Python worker with that model (mock, mlx_whisper, parakeet_mlx, ...)
//   - "mock"         -> alias for "python:mock"
//   - anything else  -> a cloud provider (deepgram, assemblyai, ...)
//
// timeout caps one worker call; long recordings on local models need minutes.
func Build(spec, workerURL string, timeout time.Duration) Engine {
	switch {
	case strings.HasPrefix(spec, "python:"):
		return NewPythonEngine(strings.TrimPrefix(spec, "python:"), workerURL, "", timeout)
	case spec == "mock":
		return NewPythonEngine("mock", workerURL, "", timeout)
	default:
		return NewCloudEngine(spec)
	}
}
