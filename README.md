# visor

visor is a go-based execution board for a telegram-first ai agent with memory, voice, scheduling, level-ups, and self-evolution hooks.

it is built to stay simple in runtime shape:
- one webhook server
- one active owner chat
- optional sidecar level-ups (email, obsidian, cloudflared)

## current status

active roadmap and milestone tracking live in:
- `visor.forge.md`

active execution tracking + handoff rules:
- `COORDINATION.md`

note:
- backlog.md layer was removed to keep execution lean

### Current: M12 — Iteration 2: optional level-ups ✅

#### Tasks
- [x] 1. Extended setup actions with optional level-up controls (`enable`, `.levelup.env`, `start`, `validate`, `health`) — `internal/setup/actions.go`, `internal/server/server.go`
- [x] 2. Added compose `up -d` runtime for currently enabled level-ups — `internal/levelup/compose_apply.go`
- [x] 3. Added enabled-levelup HTTP health verification helper — `internal/levelup/health.go`
- [x] 4. Added Forgejo remote sync setup action for onboarding flow — `internal/server/server.go`
- [x] 5. Added tests for compose apply args and env-template expansion — `internal/levelup/compose_apply_test.go`, `internal/levelup/health_test.go`

#### Status
- M1–M8a: done
- M10 Iteration 1: done
- M10 Iteration 2: done
- M10 Iteration 3: done
- M12 Iteration 1: done
- M12 Iteration 2: done

## first 10 minutes

```bash
# 1) clone + enter
git clone <your-repo-url> visor
cd visor

# 2) set minimum env (required)
export TELEGRAM_BOT_TOKEN="<bot-token>"
export USER_PHONE_NUMBER="<telegram-chat-id>"

# 3) choose agent backend (quick smoke test)
export AGENT_BACKEND="echo"

# 4) run
go run .

# 5) verify server
curl -s http://localhost:8080/health
```

if you want full ubuntu walkthroughs:
- english: `docs/ubuntu-24-noob-install.md`
- deutsch: `docs/ubuntu-24-noob-install.de.md`

## architecture (short)

- `main.go`: startup wiring (config, observability, agent backend selection)
- `internal/server`: telegram webhook handling + action execution (skills/scheduler/levelup/email)
- `internal/agent`: backend adapters (`echo`, `pi`, `claude`) + queueing + failover registry
- `internal/memory`: local memory store/search
- `internal/voice`: stt + tts wiring
- `internal/scheduler`: scheduled task persistence and dispatch
- `skills/`: runtime skill scripts/manifests
- `levelups/`: optional extensions with manifest + compose overlays

## config reference

full variable reference (required vs optional):
- `docs/config-reference.md`

level-up env template:
- `.levelup.env.example`

## operations

runbook for local run, logs, troubleshooting, and update flow:
- `docs/operations.md`

observability-specific troubleshooting:
- `docs/observability-troubleshooting.md`

## contributing

see `CONTRIBUTING.md`.

## license

MIT, see `LICENSE`.
