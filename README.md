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

### Current: M12 — Iteration 3: personality + finish ✅

#### Tasks
- [x] 1. Added setup action fields for personality/test/summary/cleanup — `internal/setup/actions.go`
- [x] 2. Added personality override + setup cleanup + summary writers — `internal/setup/finalize.go`
- [x] 3. Added setup execute hooks for Telegram test message + summary output + CLAUDE cleanup — `internal/server/server.go`
- [x] 4. Expanded first-run setup guidance context to include finish flow — `internal/setup/detect.go`
- [x] 5. Added tests for personality override + setup summary write — `internal/setup/finalize_test.go`

#### Status
- M1–M8a: done
- M10 Iteration 1: done
- M10 Iteration 2: done
- M10 Iteration 3: done
- M12 Iteration 1: done
- M12 Iteration 2: done
- M12 Iteration 3: done

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
