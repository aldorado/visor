# visor execution board

current focus: *m0 / iteration 1* — level-up framework

## status
- chunk in this commit: add admin command for list/enable/disable level-ups
- state: done

## granular todos
- [x] define level-up manifest schema (`levelup.toml`) and runtime rules
- [x] add `.levelup.env` loader with required-key validation (fail fast)
- [x] implement compose assembly (base + selected overlays)
- [x] add admin command: list/enable/disable level-ups
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
- `internal/levelup/manifest.go` -> discover + parse `levelup.toml`
- `internal/levelup/state.go` -> persist enabled level-ups (`data/levelups/enabled.json`)
- `internal/levelup/admin.go` -> list/enable/disable operations
- `internal/levelup/admin_test.go` -> command behavior tests
- `cmd/visor-admin/main.go` -> CLI admin entrypoint (`levelup list|enable|disable`)
- `go.mod`, `go.sum` -> TOML parser dependency
- `README.md` -> execution board + task states (shared, small edits only)

## next checkpoint question
continue with m0/iteration1 chunk 5 (`compose builder validation via docker compose config`)?

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
