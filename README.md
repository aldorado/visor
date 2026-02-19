# visor execution board

current focus: *m0b / iteration 1* — logging contract + modes

## status
- iteration state: done ✅
- reporting mode: per full iteration

## m0b iteration 1 todos
- [x] define global logging schema (`component`, `function`, `request_id`, `message`, attrs)
- [x] implement logger package with normal/verbose mode (`LOG_LEVEL`, `LOG_VERBOSE`)
- [x] add helper wrappers for consistent component/function naming
- [x] add panic/recover middleware with compact traceback logging

## m0b usage
- normal mode: `LOG_LEVEL=info`, `LOG_VERBOSE=false`
- verbose mode: `LOG_LEVEL=debug`, `LOG_VERBOSE=true`

## file touch map (m0b it1)
- `internal/observability/logger.go` -> slog setup, schema helpers, component/function fields
- `internal/observability/context.go` -> request-id context helpers
- `internal/observability/recover.go` -> panic/recover middleware + compact traceback
- `internal/server/server.go` -> structured lifecycle logs + recover middleware wiring
- `internal/agent/queue.go` -> queue processing logs with component naming
- `main.go`, `internal/config/config.go` -> logging mode config + initialization

## next checkpoint question
continue with *m0b / iteration 2* (full request lifecycle visibility + request-id middleware)?

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
