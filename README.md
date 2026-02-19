# visor execution board

current focus: *m6 / iteration 1* — skill runtime

## status
- iteration state: done ✅
- reporting mode: per full iteration

## m6 iteration 1 todos
- [x] skill.toml manifest format: name, description, triggers (regex), run command, dependencies, level_ups, timeout
- [x] skill loader: LoadAll scans directories, parses TOML, compiles trigger regexes
- [x] skill executor: sandboxed subprocess, configurable timeout, stdout/stderr capture
- [x] context passing: env vars (VISOR_USER_MESSAGE, VISOR_CHAT_ID, etc.) + JSON on stdin
- [x] trigger matching: case-insensitive regex, MatchAll returns all matching skills
- [x] 13 tests: load, defaults, missing fields, bad triggers, match, matchAll, execute, stdin, non-zero exit, timeout

## m0b usage
- normal mode: `LOG_LEVEL=info`, `LOG_VERBOSE=false`
- verbose mode: `LOG_LEVEL=debug`, `LOG_VERBOSE=true`
- otel disabled: `OTEL_ENABLED=false`
- otel enabled (signoz):
  - `OTEL_ENABLED=true`
  - `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318`
  - `OTEL_SERVICE_NAME=visor`
  - `OTEL_ENVIRONMENT=dev`

## m0b log reading quickstart
- systemd: `journalctl -u visor -f`
- docker: `docker logs -f <visor-container>`
- local run: stdout/stderr stream from process

sample structured line:
`time=... level=INFO msg="webhook message processed" component=server function=server.(*Server).handleWebhook request_id=... trace_id=... span_id=... chat_id=... backend=pi`

## docs
- `docs/signoz-setup.md`
- `docs/observability-troubleshooting.md`

## file touch map (m6 it1)
- `internal/skills/skill.go` -> Manifest struct, Skill loader, trigger matching
- `internal/skills/executor.go` -> subprocess execution, context injection (env + stdin JSON), timeout handling
- `internal/skills/skill_test.go` -> 13 tests covering load/match/execute
- `.claude/` -> new claude backend config (CLAUDE.md + 18 skills)
- `.pi/skills/levelup-creator/` -> synced missing skill from skills/
- `README.md`, `visor.forge.md` -> m6 iteration-1 progress tracking

## next checkpoint question
continue with *m6 / iteration 2* (agent-authored skills, discovery, auto-trigger)?

planning note:
- m9 (multi-pi-subagent orchestration) was added to forge blueprint for later execution.

---

# M1: skeleton — webhook + echo

*M1 complete* ✅

## m1 status
- iteration 1: done (project setup)
- iteration 2: done (telegram integration)
- iteration 3: done (echo bot)

## m1 iteration 3 todos
- [x] wire webhook → parse → echo response → send
- [x] text echo, voice acknowledgment, photo acknowledgment

---

# M2: agent process manager

*M2 complete* ✅

## m2 status
- iteration 1: done (agent interface + queue)
- iteration 2: done (pi adapter)
- iteration 3: done (claude code adapter)

## m2 iteration 3 todos
- [x] claude code adapter: process-per-request `claude -p --output-format stream-json`
- [x] parse stream-json events: `assistant` (text blocks), `result` (error check)
- [x] 5min timeout, proper process cleanup
- [x] AGENT_BACKEND=claude wired in main.go

---

# M3: memory system

current focus: *m3 / iteration 2* — embeddings + search

## m3 status
- iteration 1: done (parquet storage + session logger)
- state: *iteration 2 done*

## m3 iteration 2 todos
- [x] OpenAI embeddings API client (single + batch)
- [x] cosine similarity search in pure Go
- [x] Search with maxResults, minResults, minSimilarity threshold
- [x] MemoryManager: Save (embed + store) and Lookup (search + format)
- [x] config: OPENAI_API_KEY, DATA_DIR
- [x] tests: 12 search/embedding tests (cosine sim, ranking, threshold, minResults, response parsing)

---

# M4: voice pipeline

current focus: *m4 / iteration 2* — text-to-speech

## m4 status
- iteration 1: done (speech-to-text)
- state: *iteration 2 done*

## m4 iteration 1 todos
- [x] Whisper STT client (multipart upload to OpenAI API)
- [x] Voice handler: download from Telegram + transcribe
- [x] Server wiring: voice messages auto-transcribed with [Voice message] tag
- [x] Graceful fallback: if no OPENAI_API_KEY, passes raw file ID
- [x] Tests: whisper response parsing (normal, empty, unicode)

## m4 iteration 2 todos
- [x] ElevenLabs TTS client (eleven_multilingual_v2, returns MP3 bytes)
- [x] Telegram SendVoice: multipart upload of audio file
- [x] Config: ELEVENLABS_API_KEY, ELEVENLABS_VOICE_ID
- [x] Voice handler: SynthesizeAndSend (TTS → send voice message)
- [x] Agent response metadata: `send_voice: true` flag via `---` separator
- [x] Server callback: parseResponse extracts send_voice, routes to TTS with text fallback
- [x] Tests: 3 elevenlabs tests, 6 parseResponse tests
