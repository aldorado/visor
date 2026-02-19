# visor execution board

current focus: *m0 / iteration 5* — cloudflared base level-up

## status
- iteration state: done ✅
- reporting mode: per full iteration

## m0 iteration 5 todos
- [x] add cloudflared base level-up manifest
- [x] add `docker-compose.levelup.cloudflared.yml`
- [x] add cloudflared env contract for first-start setup

## cloudflared setup for new users
- `CLOUDFLARED_TUNNEL_TOKEN`:
  generate in cloudflare zero trust:
  `networks -> tunnels -> create tunnel -> docker connector -> copy token`
  then paste it into `.levelup.env`.
- `CLOUDFLARED_METRICS_PORT`:
  this is a local host port for cloudflared metrics/health endpoint.
  pick a free local port (default `20241` is fine).
  if that port is already used, change it to another free one (example: `20242`).

## file touch map (iteration 5)
- `levelups/cloudflared/levelup.toml` -> base connectivity level-up manifest
- `docker-compose.levelup.cloudflared.yml` -> tunnel sidecar overlay
- `.levelup.env.example` -> tunnel token + metrics envs
- `visor.forge.md` -> M0 iteration-5 tracking/checklist
- `visor.md` -> architecture direction update (cloudflared as base level-up)

## next checkpoint question
before m1/m3: run new *m0b observability baseline* (human-readable lifecycle logs + verbose mode + otel/sigNoz)?

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
