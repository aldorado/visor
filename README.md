# visor execution board

current focus: *m0 / iteration 1* — level-up framework

## status
- chunk in this commit: define `levelup.toml` manifest format
- state: done

## granular todos
- [x] define level-up manifest schema (`levelup.toml`) and runtime rules
- [ ] add `.levelup.env` loader with required-key validation (fail fast)
- [ ] implement compose assembly (base + selected overlays)
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
- `docs/levelup-manifest.md` -> manifest schema + rules (agent-a)
- `levelups/himalaya/levelup.toml` -> reference level-up manifest (agent-a)
- `levelups/obsidian/levelup.toml` -> standard level-up manifest (agent-a)
- `README.md` -> execution board + task states (shared, small edits only)

## next checkpoint question
continue with m0/iteration1 chunk 2 (`.levelup.env` loader + strict required-env validation)?

---

# M1: skeleton — webhook + echo

current focus: *m1 / iteration 1* — project setup

## m1 status
- state: *iteration 1 done*

## m1 granular todos
- [x] init Go module (`go.mod`)
- [x] create project structure: `main.go`, `internal/platform/`, `internal/config/`, `internal/server/`
- [x] config loader from env vars (TELEGRAM_BOT_TOKEN, USER_PHONE_NUMBER, PORT)
- [x] HTTP server with `/webhook` and `/health` routes

## m1 file touch map
| task | files |
|------|-------|
| go module | `go.mod` |
| project structure | `main.go`, `internal/platform/`, `internal/config/`, `internal/server/` |
| config loader | `internal/config/config.go` |
| http server | `main.go`, `internal/server/server.go` |
