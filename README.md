# visor execution board

current focus: *m0 / iteration 5* — cloudflared base level-up

## status
- iteration state: done ✅
- reporting mode: per full iteration

## m0 iteration 5 todos
- [x] add cloudflared base level-up manifest
- [x] add `docker-compose.levelup.cloudflared.yml`
- [x] add cloudflared env contract for first-start setup

## file touch map (iteration 5)
- `levelups/cloudflared/levelup.toml` -> base connectivity level-up manifest
- `docker-compose.levelup.cloudflared.yml` -> tunnel sidecar overlay
- `.levelup.env.example` -> tunnel token + metrics envs
- `visor.forge.md` -> M0 iteration-5 tracking/checklist
- `visor.md` -> architecture direction update (cloudflared as base level-up)

## next checkpoint question
continue with *m1 deploy/e2e* or jump straight to *m3 memory system*?

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

current focus: *m3 / iteration 1* — parquet storage

## m3 status
- state: *iteration 1 done*

## m3 iteration 1 todos
- [x] parquet-go dependency added (v0.27.0)
- [x] Memory schema: id, text, embedding (float32 list), created_at
- [x] Store with append-by-new-chunk strategy
- [x] ReadAll: load all chunks, sort by created_at
- [x] FilterByDate: range filter on unix millis
- [x] Compact: merge all chunks into single file
- [x] Session logger: JSONL per session, ReadAllSessions
- [x] Tests: 7 store tests + 4 session tests (all pass)
