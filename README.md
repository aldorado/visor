# visor

visor is a host-native go runtime for a personal ai agent.

it handles the *runtime body*: telegram webhook, agent routing, memory, voice, scheduler, and skills.

the model backend is swappable (`pi`, `claude`, `gemini`, `echo`).

## project status

core runtime is implemented.

done: `M0, M0b, M1, M2, M4, M5, M6, M7, M8, M8a, M10, M11, M12`  
in progress/open: `M3` (remote memory sync), `M9` (multi-subagent orchestration)

source of truth: `visor.forge.md`

## quickstart (local smoke test)

```bash
git clone <your-repo-url> visor
cd visor

go build -o bin/visor .

export TELEGRAM_BOT_TOKEN="<bot-token>"
export USER_PHONE_NUMBER="<telegram-chat-id>"
export AGENT_BACKEND="echo"

./bin/visor
curl -s http://localhost:8080/health
```

## first-time guided setup

want full guided onboarding on ubuntu 24?

- english: `docs/ubuntu-24-noob-install.md`
- deutsch: `docs/ubuntu-24-noob-install.de.md`

## architecture (short)

- `main.go` startup wiring
- `internal/server` webhook + action execution
- `internal/agent` adapters + queue + failover
- `internal/memory` parquet memory + semantic lookup
- `internal/voice` whisper + elevenlabs
- `internal/scheduler` reminders/recurrence
- `skills/` runtime skills
- `.pi/`, `.claude/`, `.gemini/` synced agent metadata + skills

## config and ops

- config reference: `docs/config-reference.md`
- env templates: `.env.example`
- operations runbook: `docs/operations.md`
- observability troubleshooting: `docs/observability-troubleshooting.md`

## contributing / license

- contributing: `CONTRIBUTING.md`
- license: `LICENSE`
