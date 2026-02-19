# visor execution board

current focus: *m0 / iteration 2* — himalaya email level-up (reference)

## status
- iteration state: done ✅
- reporting mode requested by user: only after full iteration (not per chunk)

## m0 iteration 2 todos
- [x] add `docker-compose.levelup.email-himalaya.yml` (*first concrete level-up shipped*)
- [x] add `docker-compose.levelup.obsidian.yml` (already present, kept as standard level-up)
- [x] define required secrets in `.levelup.env.example` (himalaya + obsidian)
- [x] implement inbound mail polling bridge into visor events
- [x] implement outbound mail send action from agent structured output
- [x] add roundtrip tests: receive email → agent sees it → send reply
- [x] add smoke test: obsidian sidecar reachable + mount persistence checks
- [x] ensure obsidian bind mounts resolve to host filesystem paths writable by visor

## file touch map (iteration 2)
- `docker-compose.levelup.email-himalaya.yml` -> first concrete email level-up
- `.levelup.env.example` -> himalaya runtime flags + polling config + secrets
- `internal/levelup/email/*` -> himalaya client, poller, action parsing, bridge, tests
- `internal/server/server.go` -> inbound email enqueue + outbound email action execution
- `internal/config/config.go` -> himalaya feature flags + poll interval config
- `internal/levelup/obsidian_smoke.go` + tests -> reachability and writable mount checks
- `internal/levelup/admin.go` -> validation flow includes obsidian mount checks

## next checkpoint question
continue with *m0 / iteration 3* (generalization docs + failure-mode docs)?

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
