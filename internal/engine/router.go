package engine

import "strings"

// Build returns an Engine for a spec:
//   - "python:<key>" -> the Python worker with that model (mock, faster_whisper, ...)
//   - "mock"         -> alias for "python:mock"
//   - anything else  -> a cloud provider (deepgram, assemblyai, ...)
func Build(spec, workerURL string) Engine {
	switch {
	case strings.HasPrefix(spec, "python:"):
		return NewPythonEngine(strings.TrimPrefix(spec, "python:"), workerURL, "")
	case spec == "mock":
		return NewPythonEngine("mock", workerURL, "")
	default:
		return NewCloudEngine(spec)
	}
}
