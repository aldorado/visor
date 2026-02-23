# operations runbook

this is the practical runbook for day-to-day operation.

## local run

```bash
export TELEGRAM_BOT_TOKEN="<bot-token>"
export USER_PHONE_NUMBER="<chat-id>"
export AGENT_BACKEND="echo"
mkdir -p bin
go build -o bin/visor .
./bin/visor
```

health check:

```bash
curl -s http://localhost:8080/health
```

## logs

local:
- stdout/stderr from `./bin/visor`

recommended for production:
- run under systemd and inspect via `journalctl -u visor -f`

## troubleshooting quick path

1. config crash on startup
   - verify `TELEGRAM_BOT_TOKEN` and `USER_PHONE_NUMBER`
   - verify `PORT` is numeric
2. no replies in telegram
   - webhook not set or wrong public url
   - bot token mismatch
   - wrong target chat id
3. voice not working
   - missing `OPENAI_API_KEY` (stt)
   - missing `ELEVENLABS_API_KEY` or `ELEVENLABS_VOICE_ID` (tts)
detailed docs:
- `docs/observability-troubleshooting.md`

## update flow

safe update sequence:

```bash
git pull --ff-only
go test ./...
go build ./...
```

if checks pass, restart your process manager (systemd/supervisor).

## release hygiene

before tagging a release:

```bash
gofmt ./...
go vet ./...
go test ./...
```

also verify:
- `README.md` matches current startup flow
- `docs/config-reference.md` is up to date
- `CHANGELOG.md` has an entry for the release
