# Visor

## Vision
A fast, compiled agent runtime in Go that serves as the "body" for swappable AI CLI backends. Visor handles everything except thinking: webhooks, memory, voice, scheduling, sessions. The AI brain (pi, claude code, gemini, etc.) plugs in via stdin/stdout RPC. Main motivation: exploit free tokens from every AI company's CLI offering.

## Key decisions
- Language: Go (bottleneck is model latency, not server perf. Go's stdlib covers HTTP/JSON/exec/crypto natively. single binary, fast iteration)
- Storage: Parquet files for memories + embeddings (portable, remote-ready — can sync to cloud/S3 later)
- Voice: API-based (OpenAI Whisper for STT, ElevenLabs for TTS — no need for local models)
- Agent interface: stdin/stdout JSON-lines RPC (each backend gets a thin adapter)
- Primary backend strategy: use pi.dev as the main runtime/auth hub; other providers are accessed through pi.dev where possible.
- Embeddings: OpenAI API (proven pattern from jarvis + ubik, keeps Go binary lean)
- Config: single TOML or env file, no YAML
- Self-evolution: visor can edit its own Go source, git commit + push, rebuild itself, and hot-restart. this is a core feature, not an afterthought.
- Email: first-class in visor (receive + send), not an afterthought skill.
- Level-ups: optional infra sidecars via Docker Compose overlays + `.levelup.env` (visor itself stays host-native, not containerized).
- Himalaya: official reference implementation of the level-up pattern (email as exemplar) and the *first* implemented level-up in the project.
- Obsidian: additional standard level-up shipped by default (knowledge workspace sidecar) via LinuxServer container.
- Skill parity bootstrap: ship visor with the same baseline skills as current ubik by copying them into `visor/skills/` as the initial pack.

## Research tasks
- [x] Investigate Claude Code RPC mode — does `claude --mode rpc` exist? what's the protocol? document stdin/stdout message format
- [x] Investigate pi CLI RPC protocol differences from what we learned in ubik — any undocumented events?
- [x] Investigate Gemini CLI — does Google offer a coding agent CLI? what RPC options exist? (lower priority — pi + claude first)
- [x] Investigate GitHub Copilot CLI — `gh copilot` capabilities, can it be used as a persistent agent? (lower priority)
- [x] Research Go libraries for Parquet read/write (parquet-go, apache arrow-go) — performance for append + scan
- [x] Research Go libraries for Telegram Bot API (telebot, gotgbot, or raw HTTP)
- [x] Check free tier limits for each CLI: tokens/day, rate limits, model access
- [x] Research Himalaya integration modes: CLI wrapping vs long-running service, IMAP IDLE support, SMTP send flow
- [x] Design compose overlay merge strategy (`docker-compose.yml` + `docker-compose.levelup.<name>.yml`) and env layering with `.levelup.env`

## Research findings log

### 2026-02-19 — Claude Code interface check (step 1)
- `claude --mode rpc` is *not* available in Claude Code `2.1.47` (checked via `claude --help`).
- Viable non-interactive integration path exists via:
  - `claude -p --output-format stream-json --verbose "<prompt>"`
- Observed stream-json output events include:
  - `system` (session/bootstrap metadata)
  - `rate_limit_event`
  - `assistant` (message payload)
  - `result` (final status/usage/cost)
- Input streaming mode exists (`--input-format stream-json`), but exact stdin event contract still needs deeper probing/docs before relying on bidirectional long-lived RPC behavior.
- Practical adapter decision for now: implement Claude adapter as *process-per-request* stream-json mode, not persistent `--mode rpc`.

### 2026-02-19 — pi RPC protocol check (step 2)
- `pi --mode rpc` is available in pi `0.53.0`.
- Prompt command contract confirmed:
  - stdin command: `{"type":"prompt","message":"..."}`
  - immediate ack event: `{"type":"response","command":"prompt","success":true|false,...}`
- Observed top-level events:
  - `agent_start`, `turn_start`, `message_start`, `message_update`, `message_end`, `turn_end`, `agent_end`
  - plus tool lifecycle events when tools are enabled: `tool_execution_start`, `tool_execution_update`, `tool_execution_end`
- Observed `assistantMessageEvent.type` values inside `message_update`:
  - text stream: `text_start`, `text_delta`, `text_end`
  - reasoning stream: `thinking_start`, `thinking_delta`, `thinking_end`
  - tool-call stream: `toolcall_start`, `toolcall_delta`, `toolcall_end`
- Error behavior confirmed:
  - invalid json / unknown command / malformed prompt emit `type:"response", success:false, error:"..."`
- Adapter implication:
  - current ubik extraction via `text_delta` + `agent_end` is still valid
  - but visor adapter should also track `response success:false` as hard failure and optionally expose tool/thinking streams for richer telemetry.

### 2026-02-19 — Gemini CLI capability check (step 3)
- Google offers a coding agent CLI via npm package `@google/gemini-cli` (checked latest `0.29.2`, preview/nightly also published).
- Local binary was not preinstalled, but `npx @google/gemini-cli --help` works and confirms headless mode support.
- Supported non-interactive output formats:
  - `text`
  - `json`
  - `stream-json`
- Official docs confirm stream-json event model for headless mode with event types:
  - `init`, `message`, `tool_use`, `tool_result`, `error`, `result`
- Auth requirement is explicit (without auth it exits with config/auth error), expecting one of:
  - `GEMINI_API_KEY`
  - `GOOGLE_GENAI_USE_VERTEXAI`
  - `GOOGLE_GENAI_USE_GCA`
- There is an `--experimental-acp` flag in CLI reference, but no stable public protocol contract found yet.
- Adapter implication:
  - Gemini is viable today as process-per-request headless adapter via `-p` + `--output-format stream-json`
  - treat ACP as experimental-only until protocol/docs stabilize.

### 2026-02-19 — architecture decision update
- product decision: prefer pi.dev as primary runtime/auth entrypoint.
- keep runtime-level switching between provider/subscription/model as a first-class feature.
- recommended policy:
  - default provider/model per task class (coding/chat/research)
  - automatic fallback on quota/rate/auth failures
  - manual override command for forced provider/model pinning per run

### 2026-02-19 — GitHub Copilot CLI check (step 4)
- legacy `github/gh-copilot` extension is deprecated (README points to new `github/copilot-cli`).
- current Copilot CLI is available as `@github/copilot` (public preview) and supports:
  - interactive mode
  - non-interactive prompt mode: `-p`
  - model selection via `--model`
  - an ACP mode via `--acp` (Agent Client Protocol server mode)
- local run via `npx @github/copilot --help` confirmed flags, including `--acp`, `-p`, `--stream`, `--allow-all-tools`.
- auth is mandatory and can come from:
  - Copilot login flow (`/login`)
  - env token (`COPILOT_GITHUB_TOKEN`, `GH_TOKEN`, `GITHUB_TOKEN`)
  - `gh auth login`
- practical adapter stance right now:
  - process-per-request integration via `-p` is viable once authenticated
  - `--acp` exists and is promising for persistence, but treat as secondary/optional until protocol/docs are validated against visor needs
  - pi.dev remains primary hub per architecture decision.

### 2026-02-19 — Go Parquet libs check (step 5)
- `parquet-go/parquet-go`:
  - explicitly positioned as high-performance Go Parquet library.
  - modern typed APIs (`GenericWriter[T]`, `GenericReader[T]`) fit visor's Go-first codebase well.
  - important note from docs: parquet files are immutable after write; true in-place update/append is not the natural pattern.
  - supports low-level row group/page access and schema conversion utilities.
  - currently pre-v1 (possible occasional breaking API changes).
- `apache/arrow-go` parquet stack (`parquet`, `file`, `pqarrow`):
  - broader ecosystem coverage (Arrow + Parquet + conversion bridge).
  - robust for interoperability and analytics pipelines.
  - heavier mental model (Arrow memory + retain/release lifecycle + extra layers) for simple persistence use cases.
- append + scan implications for visor memory:
  - safest append strategy is *chunked immutable files* (new parquet chunks/row-groups) plus periodic compaction/merge, not in-place mutation.
  - scan path should leverage row-group metadata and chunk-level filtering before full row decode.
- recommendation for visor v1:
  - primary writer/reader: `parquet-go/parquet-go`
  - architecture pattern: append-by-new-file (or append-by-new-row-group in new output artifact), then background compaction
  - keep Arrow integration optional for export/interchange, not core runtime dependency.

### 2026-02-19 — Telegram Go libs check (step 6)
- `telebot` (`tucnak/telebot`):
  - mature and widely used framework with concise routing/middleware API.
  - high ergonomics for quick bot feature delivery.
- `gotgbot` (`PaulSonOfLars/gotgbot/v2`):
  - code-generated API wrapper with strong parity to Telegram Bot API docs.
  - zero third-party dependency style and explicit updater/dispatcher model.
- `go-telegram/bot`:
  - zero-dependency framework, explicitly tracks latest Bot API versions (README currently references 9.4).
  - clean webhook + manual `ProcessUpdate` path.
- Raw HTTP (stdlib only):
  - max control, minimal dependency surface, easiest long-term runtime stability for a self-evolving host process.
  - but requires more internal boilerplate (routing/update parsing/retries/types).
- recommendation for visor v1:
  - keep *raw HTTP as primary* (as already intended in architecture) for minimal moving parts.
  - optionally add thin compatibility adapter layer later if framework migration becomes attractive.

### 2026-02-19 — free-tier/rate-limit check across CLIs (step 7)
- Decision context: pi.dev remains the primary runtime/auth hub; other CLIs are secondary adapters.
- Gemini CLI (well-documented quotas):
  - Google login (Code Assist Individuals): 1000 requests/day, 60 requests/min, Gemini family routing.
  - Unpaid Gemini API key: 250 requests/day, 10 requests/min, Flash-only.
  - Paid tiers increase per-user daily/minute limits; pay-as-you-go available via API key/Vertex.
- Claude Code:
  - explicit static free-tier numbers are not exposed directly in CLI help output.
  - runtime emits `rate_limit_event` with machine-readable fields (`rateLimitType`, `resetsAt`, overage flags), which is enough for dynamic routing decisions.
  - practical stance: treat Claude limits as runtime-discovered rather than hardcoded constants.
- GitHub Copilot CLI:
  - requires active Copilot subscription/auth.
  - docs/README indicate monthly premium-request accounting, but precise allowance depends on plan/subscription context.
- pi.dev hub strategy implications:
  - do not hardcode provider limits except where official and stable (Gemini values above).
  - keep a runtime quota registry fed by real events/errors (rate-limit, quota-exceeded, auth-failed).
  - route selection policy should prioritize: health -> quota headroom -> cost profile -> requested model.

### 2026-02-19 — Himalaya integration modes check (step 8)
- Himalaya is a robust email *CLI* (IMAP + SMTP + multiple backends) with JSON output support (`--output json`), which is suitable for wrapper integration.
- CLI command surface (from source) is command-oriented (`account`, `folder`, `envelope`, `message`, etc.) and does not expose a dedicated long-running daemon/service mode in current core commands.
- No explicit `idle`/`watch`/`notify` command was found in current CLI command tree, so push-style IMAP IDLE is not exposed as a first-class CLI workflow.
- SMTP send flow is cleanly represented in command layer (`message send`, `template send`) and internally calls `send_message_then_save_copy`, matching expected "send + sent copy" behavior.
- Integration recommendation for visor level-up:
  - mode A (v1): *CLI wrapping* as default (spawn short-lived himalaya commands for list/read/send)
  - mode B (later): optional custom sidecar bridge service built by us if we need push/stream semantics
  - inbound strategy in v1: polling envelopes/messages with state checkpoint, not IMAP IDLE.

### 2026-02-19 — compose overlay + env layering design (step 9)
- Compose merge order should be deterministic: base first, then level-up overlays in declared order (`-f base -f overlay1 -f overlay2 ...`).
- Merge semantics to rely on:
  - single-value fields replace previous values
  - selected list fields concatenate
  - map-like fields (eg `environment`, `labels`) merge with later file taking precedence on key conflicts
- Path safety rule: all relative paths must be resolved against the *base* compose file directory.
- Validation rule: always render merged model via `docker compose config` before up/down operations.
- Env layering design for visor level-ups:
  - source layers: `.env` (base) + `.levelup.env` (level-up specific) + process env overrides
  - required level-up keys declared in `levelup.toml`; fail fast if missing
  - secrets stay out of compose YAML; referenced through env and/or compose secrets.
- Scope boundary stays strict:
  - compose manages sidecars only
  - visor host process remains outside compose
  - himalaya email level-up is the first and canonical example integration.

### 2026-02-19 — Obsidian standard level-up added
- Added an additional default level-up overlay for Obsidian based on LinuxServer `docker-obsidian`.
- Compose file target: `docker-compose.levelup.obsidian.yml`.
- Env contract added for `.levelup.env`: PUID/PGID/TZ, optional basic auth (CUSTOM_USER/PASSWORD), title/subfolder, host paths, and host ports.
- Positioning: Obsidian is a standard level-up shipped alongside Himalaya (not replacing Himalaya as first canonical example).

### 2026-02-19 — Skill parity + filesystem mount update
- Copied the full current ubik skill pack from `/root/code/ubik/.pi/skills/` to `/root/code/visor/skills/` as visor bootstrap parity.
- Obsidian level-up explicitly uses host bind mounts for `/config` and `/vault` so host-native visor can directly read/write vault files.

## Milestones

### M0: host-native runtime boundary + level-up foundation + native email baseline
Lock the boundary first: visor is host-native, compose is sidecars-only. Then use Himalaya email as canonical first level-up.

#### Iteration 1: level-up framework
- [x] Define level-up manifest format (`levelup.toml`) with name, compose overlay file, required env keys, healthcheck
- [x] Add loader for `.levelup.env` with strict validation (fail fast if required vars missing)
- [x] Implement compose assembly strategy (base + selected overlays)
- [x] Add CLI/admin command to list/enable/disable level-ups
- [x] Compose merge rules in runtime builder:
  - base file first, then overlays in declared order
  - never include visor service in compose (sidecars only)
  - enforce all relative paths resolved from base compose file directory
  - verify final model with `docker compose config` before apply

#### Iteration 2: Himalaya email level-up (reference)
- [x] Add `docker-compose.levelup.email-himalaya.yml` (*first concrete level-up shipped*)
- [x] Add `docker-compose.levelup.obsidian.yml` (standard knowledge workspace level-up)
- [x] Define required secrets in `.levelup.env.example` (IMAP/SMTP host, user, app-password, tls flags + Obsidian auth/paths/ports)
- [x] Implement inbound mail polling/IDLE bridge into visor events
- [x] Implement outbound mail send action from agent structured output
- [x] Add roundtrip tests: receive email → agent sees it → send reply
- [x] Add smoke test: Obsidian sidecar is reachable and persists vault/config mounts
- [x] Ensure Obsidian bind mounts resolve to host filesystem paths accessible by visor runtime

#### Iteration 3: generalization docs
- [x] Write "how to build a level-up" guide using Himalaya as template
- [x] Add second toy level-up (minimal stub) to prove pattern is generic
- [x] Add failure-mode docs (missing env, container down, auth error)

#### Iteration 4: self-authored level-up skill
- [x] Add visor skill `levelup-creator` that lets visor create its own level-ups end-to-end
- [x] Require this skill to use M0 docs (`levelup-authoring`, `levelup-failure-modes`, `levelup-manifest`)
- [x] Enforce validation + docs sync + iteration-scoped commit in the skill workflow

### M1: skeleton — webhook + echo
Get a Go binary that receives Telegram webhooks and echoes messages back.

#### Iteration 1: project setup
- [x] Init Go module in `/root/code/visor`
- [x] Basic project structure: `main.go`, `internal/platform/`, `internal/agent/`
- [x] Load config from env vars (TELEGRAM_BOT_TOKEN, USER_PHONE_NUMBER, PORT)
- [x] HTTP server with /webhook and /health routes

#### Iteration 2: telegram integration
- [x] Parse Telegram webhook payloads (text, voice, image, reactions)
- [x] Send text responses via Telegram Bot API
- [x] Webhook signature verification
- [x] Auth check: drop messages not from USER_PHONE_NUMBER
- [x] Message dedup (in-memory set with TTL)

#### Iteration 3: echo bot
- [x] Wire webhook → parse → echo response → send
- [ ] Deploy and test end-to-end on telegram

### M2: agent process manager
Persistent CLI agent process with RPC over stdin/stdout.

#### Iteration 1: agent interface
- [x] Define Go interface: `Agent { SendPrompt(ctx, prompt) -> (response, error) }`
- [x] Implement process manager: spawn, restart on crash, periodic restart (configurable)
- [x] Message queue: if agent is busy, queue incoming messages

#### Iteration 2: pi adapter
- [x] Implement pi CLI adapter (`pi --mode rpc`)
- [x] Handle JSON-lines protocol: send `{ type: "prompt", message: "..." }`
- [x] Collect `text_delta` events, resolve on `agent_end`
- [x] Handle errors, timeouts (configurable per-prompt timeout)

#### Iteration 3: claude code adapter
- [x] Research Claude Code RPC protocol (depends on research task)
- [x] Implement adapter following same interface
- [x] Test switching between pi and claude via config

### M3: memory system
Persistent memory with semantic search. All data in Parquet files — portable, remote-ready, can sync to S3/cloud later.

#### Iteration 1: parquet storage
- [x] Parquet read/write in Go (parquet-go library or apache arrow)
- [x] Memories schema: id, text, embedding (float32 array), created_at
- [x] Sessions schema: id, user_id, role, content, timestamp
- [x] Store files in `data/memories.parquet` and `data/sessions/` (JSONL per session)
- [x] Basic CRUD: append to parquet, read all, filter by date

#### Iteration 2: embeddings + search
- [ ] Call OpenAI embeddings API to generate vectors
- [ ] Store embeddings as float32 arrays in parquet columns
- [ ] Implement cosine similarity search in Go (load parquet → scan embeddings → rank)
- [ ] Memory save: agent response includes memories_to_save → auto-embed and store
- [ ] Memory lookup: before each prompt, search relevant memories and inject as context

#### Iteration 3: remote sync (future)
- [ ] Design sync protocol: local parquet ↔ remote storage (S3, R2, or custom)
- [ ] Incremental sync: only push new rows, not full file rewrites

### M4: voice pipeline
Transcription + text-to-speech.

#### Iteration 1: speech-to-text
- [ ] Download voice message from Telegram
- [ ] Send to OpenAI Whisper API for transcription
- [ ] Feed transcribed text to agent as normal message with [Voice message] tag

#### Iteration 2: text-to-speech
- [ ] ElevenLabs TTS API integration
- [ ] Agent can request voice response (send_voice flag)
- [ ] Send audio file back via Telegram

### M5: scheduling + cron
In-process scheduler for reminders and recurring tasks.

#### Iteration 1: scheduler
- [ ] In-process cron scheduler (no system crontab dependency)
- [ ] Persist scheduled tasks to disk as JSON (survives restarts)
- [ ] Support one-shot and recurring schedules
- [ ] On trigger: send prompt to agent with context about what was scheduled

#### Iteration 2: agent integration
- [ ] Agent can create/modify/delete scheduled tasks via structured output
- [ ] List upcoming tasks on request

### M6: skills system
Agent can create, edit, import, and execute skills autonomously, and request level-up enablement when infra dependencies are needed.

#### Iteration 1: skill runtime
- [x] Bootstrap parity pack copied from ubik into `visor/skills/`
- [ ] Define skill format: executable scripts in `skills/` directory (shell, python, etc.)
- [ ] Skill manifest: each skill has a `skill.toml` with name, description, trigger patterns, dependencies
- [ ] Skill executor: visor runs skills in a sandboxed subprocess, captures stdout/stderr
- [ ] Pass context to skills via env vars or stdin (user message, chat history summary, etc.)

#### Iteration 2: agent-authored skills
- [ ] Agent can create new skills via structured output (writes script + manifest)
- [ ] Agent can edit existing skills (modify script or manifest)
- [ ] Agent can delete skills
- [ ] Skill discovery: agent gets list of available skills in its system prompt context
- [ ] Auto-trigger: visor matches incoming messages against skill trigger patterns, suggests or auto-runs
- [ ] Skill dependency handshake: skill can declare required level-up(s); visor prompts for enablement if missing

#### Iteration 3: skill import
- [ ] Import skills from git repos or URLs
- [ ] Dependency resolution: skills can declare required tools/packages
- [ ] Version tracking: git hash or semver per skill

### M7: multi-backend + auto-switch
Rotate between AI backends based on availability.

#### Iteration 1: backend registry
- [ ] Config: list of backends with priority order
- [ ] Health check per backend (is the CLI installed? does auth work?)
- [ ] Select highest-priority healthy backend

#### Iteration 2: auto-failover
- [ ] Detect rate limit / quota errors from backend
- [ ] Auto-switch to next available backend
- [ ] Log which backend is active
- [ ] Notify user on backend switch (optional)

### M8: self-evolution
Visor's agent can modify visor's own source code, commit, push, rebuild, and restart — fully autonomous.

#### Iteration 1: self-edit pipeline
- [ ] Agent has read/write access to visor's source directory (its own codebase)
- [ ] Structured output includes `code_changes: true` flag when source was modified
- [ ] On `code_changes: true`: visor sends the response to the user FIRST, then triggers the build/restart pipeline
- [ ] Pipeline: `git add -A && git commit -m "..." && git push` (agent provides commit message)

#### Iteration 2: self-rebuild + restart
- [ ] After commit: run `go build -o visor-new .` to compile the new binary
- [ ] If build fails: notify user with error, rollback the commit (`git reset HEAD~1`), keep running old binary
- [ ] If build succeeds: replace the running binary with the new one
- [ ] Graceful restart: finish in-flight requests, shut down agent process, exec() into new binary (or use a supervisor)
- [ ] Supervisor approach: visor spawns itself as a child, parent watches and restarts on exit. On self-update, child exits with special code, parent replaces binary and respawns.

#### Iteration 3: safety rails
- [ ] Pre-build validation: run `go vet` and basic checks before committing
- [ ] Keep last N working binaries as rollback (e.g. `visor.bak.1`, `visor.bak.2`)
- [ ] If new binary crashes within 30s of startup: auto-rollback to previous binary
- [ ] Log all self-modifications to a dedicated changelog (who changed what, when, which agent backend)
- [ ] User can disable self-evolution via config flag

## Stack / tech
- Language: Go 1.22+
- HTTP: net/http (stdlib) or chi router
- Memory storage: Parquet files via parquet-go (remote-ready, portable)
- Telegram: raw HTTP calls (keep it simple, no heavy SDK)
- Voice: OpenAI Whisper API + ElevenLabs API (HTTP calls)
- Embeddings: OpenAI API
- Config: env vars (godotenv for .env loading)
- Skills: executable scripts in `skills/` dir, managed by agent

## Open questions
- Skill sandboxing: how strict? Docker/nsjail or just subprocess with timeout?
- Should skills be able to call other skills (skill chaining)?

#forge #visor #go #project

> promoted from [ideas/visor-rewrite](../ideas/visor-rewrite.md)
