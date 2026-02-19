# visor execution board

current focus: *m0 / iteration 3* — level-up generalization docs

## status
- iteration state: done ✅
- reporting mode: per full iteration

## m0 iteration 3 todos
- [x] write "how to build a level-up" guide using himalaya as template
- [x] add second toy level-up (minimal stub) to prove pattern is generic
- [x] add failure-mode docs (missing env, container down, auth error)

## file touch map (iteration 3)
- `docs/levelup-authoring.md` -> end-to-end authoring guide (himalaya as reference)
- `docs/levelup-failure-modes.md` -> operational troubleshooting playbook
- `levelups/echo-stub/levelup.toml` -> toy generic level-up manifest
- `docker-compose.levelup.echo-stub.yml` -> toy generic overlay
- `.levelup.env.example` -> toy level-up env contract

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
