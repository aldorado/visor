# visor execution board

current focus: *m8 / iteration 1* — self-edit pipeline

## status
- iteration state: done ✅
- reporting mode: per full iteration

## m8 iteration 1 todos
- [x] selfevolve.Manager: git add -A, commit, push pipeline with change detection
- [x] Config: SELF_EVOLUTION_ENABLED, SELF_EVOLUTION_REPO_DIR, SELF_EVOLUTION_PUSH env vars
- [x] parseResponse: extract code_changes + commit_message from agent response metadata
- [x] Server wiring: on code_changes:true, sends response first, then triggers self-evolution async
- [x] Tests: updated parseResponse tests (7 total), config clearEnv for new vars

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

## file touch map (m8 it1)
- `internal/selfevolve/manager.go` -> Manager, Apply, hasGitChanges, run (git add/commit/push pipeline)
- `internal/config/config.go` -> SelfEvolutionEnabled, SelfEvolutionRepoDir, SelfEvolutionPush fields
- `internal/config/config_test.go` -> clearEnv updated for self-evolution vars
- `internal/server/server.go` -> parseResponse returns responseMeta (code_changes, commit_message), runSelfEvolution async trigger
- `internal/server/server_test.go` -> 7 parseResponse tests updated for responseMeta struct
- `README.md`, `visor.forge.md` -> m8 iteration-1 progress tracking

## next
*M8-I1 complete* ✅ — continue with *M8-I2* (self-rebuild + restart)

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
