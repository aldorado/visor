# visor forge

## vision
visor is a host-native go runtime for chat agents.
it handles the body: webhooks, memory, voice, scheduling, and skills.
the brain is swappable (pi/claude/gemini via adapters).

goal: one stable runtime, many model backends, clean ops, minimal friction.

## current status (sorted)

### completed milestones (recap)

#### m0 — host-native boundary + baseline ✅
visor stays outside docker.

#### m0b — observability baseline ✅
structured logging is rolled out across runtime paths.
request/agent lifecycle logs are in place.
otel + signoz export is integrated and toggleable via env.

#### m1 — webhook + echo skeleton ✅
telegram webhook handling, auth, dedupe, send path, and health route are running.

#### m2 — agent process manager ✅
unified agent interface, queueing, process supervision, pi + claude adapters implemented.

#### m3 — memory system (partial) 🟡
parquet-based storage, embeddings, and semantic lookup are done.
remote sync design/implementation is still open (see open milestones).

#### m4 — voice pipeline ✅
whisper transcription + elevenlabs tts are integrated end-to-end.

#### m5 — scheduling + cron ✅
in-process scheduler with persistence is live.
agent-driven create/update/delete/list works, including quick actions (`done`, `snooze`, `reschedule`).

#### m6 — skills system ✅
skill runtime, agent-authored skill edits, import flow, and dependency handshake are implemented.

#### m7 — multi-backend switching ✅
backend registry + health checks + auto failover are implemented.

#### m8 — self-evolution ✅
self-edit, commit/push, rebuild, restart, and rollback safety rails are implemented.

#### m8a — release hardening ✅
repo hygiene, docs polish, release checklist, and local quality gates are in place.

#### m9 — multi-subagent orchestration ⏳
planned but not implemented yet (manual fan-out first, automation later).

#### m10 — reverse proxy ✅
proxy and dynamic subdomain routing are implemented.

#### m11 — forgejo git ✅
forgejo sidecar + bootstrap + git remote/push workflow are implemented.

#### m12 — interactive first-run setup ✅
guided setup flow (env collection, validation, finish summary) is implemented.

## open milestones

### m3 iteration 3 — remote memory sync
- define sync contract (local parquet ↔ remote storage)
- incremental sync strategy (new rows/chunks only)
- conflict/recovery strategy

### m9 — multi-subagent orchestration
- manual trigger path + bounded parallel fan-out/fan-in
- station/domain model with ranked model fallback config (`config/subagent-stations.json`)
- reliability, observability, timeout handling
- optional auto-orchestration policy later

## next focus
1) finish m3 remote sync design + first implementation slice
2) start m9 iteration 1 (manual orchestration path)

#forge #visor #go #project
