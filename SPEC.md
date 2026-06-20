# stt-service — spec & decisions

Simple English. This file records the locked decisions and the plan.

## Goal

One reusable speech-to-text backend for all my projects:
- macOS voice-input app (like Glaido): hotkey → speak → text typed into any app.
- Telegram bot: send a voice message → get text.
- English coach: send a recording → get the transcript to analyze.

I record in **English and Russian**. I want it to run **locally** (private,
on-device) and also **deploy** cheaply.

## Locked decisions

1. **Split by responsibility.**
   - **Go** = the service: HTTP API, queue, worker pool, job store, supervision.
     This is the part with real engineering and the main Go learning.
   - **Python** = the models: a thin worker, `audio → text`. The only Python in
     the serving path. Same engine code the eval uses.
2. **Engines behind one interface** (`internal/engine`). Impls: the Python
   worker (local models) and cloud APIs (Deepgram, …). New transport or provider
   = new impl, not a rewrite.
3. **Transport = HTTP+JSON now, behind the interface.** Move to **gRPC** later
   for real-time/streaming voice. Note: gRPC is only the transport — real-time
   also needs a streaming mode on the model side, not just a new transport.
4. **Two deploy profiles, same Go binary.**
   - **local**: Go + Python worker on the Mac (MLX / faster-whisper).
   - **production**: Go + cloud APIs only. Tiny image, scale-to-zero, fast cold start.
   - Heavy ML models are **not** shipped to the scale-to-zero cloud: MLX is
     Apple-Silicon only, and loading a Whisper model makes cold start seconds,
     not ~150ms. Self-hosting models for remote use = a separate always-on box
     (or a serverless-GPU provider as a "cloud engine") — a later, deliberate
     cost choice.
5. **Don't embed Python in Go.** No cgo/libpython. The worker is a long-lived
   process (model loaded once); Go talks to it over localhost HTTP and can
   **supervise** it (start / health-check / restart) for a one-command local run.
6. **Queue behind an interface.** Start with an in-process worker pool (no
   infra). Swap to **asynq/Redis** when deploying or scaling to many nodes.

## Components

```
internal/
  config/   env + run profile (local|production)
  engine/   Engine interface; python (HTTP) + cloud (stub) impls; router
  job/      job model + states (queued|running|done|failed)
  queue/    Queue interface + in-process worker pool
  store/    Store interface + in-memory impl (SQLite/Postgres later)
  api/      HTTP handlers: sync transcribe, async job submit/poll, health
  worker/   optional supervisor for the python process
cmd/server/ wire it all up

../py-stt/  (separate repo — Python, uv, 3.11)
  stt_engines/  engine adapters (shared with eval)
  stt_worker/   FastAPI: POST /transcribe, GET /health
  stt_eval/     offline harness → MLflow
```

## Queue / concurrency

- `POST /v1/jobs` enqueues and returns a `job_id`; bounded workers process so
  several devices at once don't overload the Mac. Poll `GET /v1/jobs/{id}`.
- `POST /v1/transcribe` is a sync fast path (skip the queue) for interactive
  dictation latency.
- Later: webhook callback instead of polling; priority lane for dictation.

## Engines

`python:<key>` routes to the worker; a bare provider name routes to a cloud
engine (Go-direct). Worker engine keys:

| key | type | EN | RU | status |
|---|---|---|---|---|
| `mock` | built-in | ✅ | ✅ | ready (test wiring) |
| `faster_whisper` | local (CPU) | ✅ | ✅ | ready |
| `mlx_whisper` | local (Apple Silicon) | ✅ | ✅ | ready |
| `whisper_cpp` | local | ✅ | ✅ | stub |
| `apple_speech` | local (built-in) | ✅ | ✅ | stub (needs Swift helper) |
| `parakeet_mlx` | local (Apple Silicon) | ✅ | ⚠️ weak | stub |
| `deepgram` / `assemblyai` / `elevenlabs` | cloud | ✅ | ✅ | stub |

## Eval (`stt_eval`)

Offline tool to pick the model: run engines on my recordings (manifest =
`s3://`/`https://`/local audio + my hand-written transcript + language), score
**similarity %** (WER, CER, char-similarity), averaged **per language**, logged
to MLflow. The eval and the worker share `stt_engines`, so "the winner" is the
same code the worker serves. Whisper accuracy is the same across runtimes (same
weights), so the eval's faster-whisper result predicts the worker's whisper.cpp/
MLX result.

## Build order / status

1. ✅ Scaffold: Go service + Python worker + shared engines + eval + tests + mock path.
2. ⬜ Record ~15–30 EN/RU clips, transcribe by hand, upload to private object storage (S3-compatible), run the eval.
3. ⬜ Real cloud engine in Go (Deepgram or AssemblyAI) for the production profile.
4. ⬜ macOS voice-input app (first client) → calls `/v1/transcribe` locally.
5. ⬜ Swap in-process queue → asynq/Redis; add gRPC + streaming for real-time.
6. ⬜ Point Telegram bot + English coach at the service.

## Open questions
- LLM cleanup step (fix casing/punctuation) — in Go, in the worker, or a client concern?
- Auth for the deployed (production) profile — API key header?
- Job durability — build the SQLite/Postgres store adapter (the production profile already refuses to boot on the in-memory store).
