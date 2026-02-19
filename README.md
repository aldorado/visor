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

current focus: *m1 / iteration 2* — telegram integration

## m1 status
- iteration 1: done (project setup)
- state: *iteration 2 done*

## m1 iteration 2 todos
- [x] parse Telegram webhook payloads (text, voice, photo, reactions)
- [x] Telegram API client (send text messages, get file URLs)
- [x] webhook signature verification (X-Telegram-Bot-Api-Secret-Token)
- [x] auth check: drop messages not from USER_PHONE_NUMBER
- [x] message dedup (in-memory set with 5min TTL + background cleanup)

## m1 file touch map
| task | files |
|------|-------|
| telegram types | `internal/platform/telegram/types.go` |
| telegram client | `internal/platform/telegram/client.go` |
| message dedup | `internal/platform/telegram/dedup.go` |
| webhook handler | `internal/server/server.go` |
| config (webhook secret) | `internal/config/config.go` |
