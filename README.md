# visor execution board

current focus: *m0 / iteration 1* — level-up framework

## status
- chunk in this commit: implement compose assembly (base + selected overlays)
- state: done

## granular todos
- [x] define level-up manifest schema (`levelup.toml`) and runtime rules
- [x] add `.levelup.env` loader with required-key validation (fail fast)
- [x] implement compose assembly (base + selected overlays)
- [ ] add admin command: list/enable/disable level-ups
- [ ] add compose builder validation via `docker compose config`

## multi-agent work split
agent-a (spec/docs)
- lock manifest schema and examples
- keep docs in sync with m0 iteration 1 scope

agent-b (runtime parser)
- implement toml parser + discovery (`levelups/*/levelup.toml`)
- return typed manifest structs + strict field validation

agent-c (ops layer)
- implement env layering and compose config validation calls
- wire list/enable/disable commands

## file touch map
- `internal/levelup/compose.go` -> compose assembly builder (base + ordered overlays)
- `internal/levelup/compose_test.go` -> assembly tests (ordering, path resolution, dedup, args)
- `docs/levelup-manifest.md` -> compose assembly/runtime rules sync
- `README.md` -> execution board + task states (shared, small edits only)

## next checkpoint question
continue with m0/iteration1 chunk 4 (`admin command: list/enable/disable level-ups`)?

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

current focus: *m2 / iteration 2* — pi adapter

## m2 status
- iteration 1: done (agent interface + queue)
- state: *iteration 2 done*

## m2 iteration 2 todos
- [x] refactor ProcessManager with real stdin/stdout pipes + process start
- [x] pi adapter: `pi --mode rpc` JSON-lines protocol
- [x] send `{"type":"prompt","message":"..."}` on stdin
- [x] collect `text_delta` events, resolve on `agent_end`
- [x] handle `response success:false` as hard failure
- [x] configurable prompt timeout (default 2min)
- [x] config: AGENT_BACKEND env var (pi/echo)
- [x] main.go: agent backend selection from config
