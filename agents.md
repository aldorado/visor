# Visor Agents

## 1) Runtime Agent (Host Core)
- Owns host-native visor process lifecycle (start/stop/restart/self-update)
- Enforces boundary: visor never runs inside Docker Compose
- Handles config loading, health checks, and structured output plumbing

## 2) Backend Router Agent (pi.dev Hub)
- Uses pi.dev as primary auth/runtime hub
- Chooses provider/subscription/model at runtime
- Applies fallback policy on rate-limit/quota/auth errors
- Tracks runtime quota signals (`rate_limit_event`, provider errors)

## 3) Messaging Agent (Telegram/WhatsApp)
- Implements raw HTTP webhook ingestion and outbound sends
- Performs auth checks, dedup, and update normalization
- Maintains platform formatting compatibility

## 4) Memory Agent (Parquet Core)
- Uses `parquet-go/parquet-go` for memory/session persistence
- Implements append-by-new-chunk strategy + periodic compaction
- Owns embedding write/read/search lifecycle

## 5) Scheduler Agent
- Manages in-process reminders/recurring jobs
- Persists schedule state on disk
- Triggers agent prompts with task context

## 6) Voice Agent
- Handles STT via Whisper and TTS via ElevenLabs
- Converts voice updates into text context and voice replies

## 7) Level-up Orchestrator Agent
- Resolves enabled level-ups and compose overlays in order
- Applies env layering (`.env` + `.levelup.env` + process env)
- Validates merged model via `docker compose config`
- Manages sidecars only (never visor core)

## 8) Email Level-up Agent (Himalaya - first exemplar)
- First concrete level-up in visor
- Integrates Himalaya via CLI wrapping (short-lived commands)
- Inbound mail via polling with checkpoints (no mandatory IMAP IDLE dependency in v1)
- Outbound mail via send commands with sent-copy semantics

## 9) Skills Agent
- Manages skill discovery, execution, and dependency checks
- Requests level-up enablement when a skill needs sidecar infra
- Supports future skill import/versioning flow
