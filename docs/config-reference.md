# config reference

this file documents runtime environment variables for visor.

## required

| variable | required | default | purpose |
|---|---|---|---|
| `TELEGRAM_BOT_TOKEN` | yes | none | telegram bot api token |
| `USER_PHONE_NUMBER` | yes | none | owner telegram chat id used for inbound/outbound routing |

## core runtime

| variable | required | default | purpose |
|---|---|---|---|
| `PORT` | no | `8080` | http listen port |
| `AGENT_BACKEND` | no | `echo` | single-backend mode (`echo`, `pi`, `claude`) |
| `AGENT_BACKENDS` | no | derived from `AGENT_BACKEND` | comma-separated priority list for auto-failover |
| `TELEGRAM_WEBHOOK_SECRET` | no | empty | optional webhook secret validation |
| `DATA_DIR` | no | `data` | runtime storage base path |
| `TZ` | no | `UTC` | timezone for natural-time scheduling/quick actions (e.g. `Europe/Vienna`) |

## ai + voice

| variable | required | default | purpose |
|---|---|---|---|
| `OPENAI_API_KEY` | no | empty | enables stt + embedding-backed features |
| `ELEVENLABS_API_KEY` | no | empty | enables tts |
| `ELEVENLABS_VOICE_ID` | no | empty | voice id for elevenlabs tts |

## email level-up (himalaya)

| variable | required | default | purpose |
|---|---|---|---|
| `HIMALAYA_ENABLED` | no | `false` | enables inbound/outbound email integration |
| `HIMALAYA_ACCOUNT` | no | `default` | himalaya account profile name |
| `HIMALAYA_POLL_INTERVAL_SECONDS` | no | `60` | inbox polling interval |
| `EMAIL_ALLOWED_SENDERS` | no | empty | comma-separated allowlist for inbound forwarding |

## logging + observability

| variable | required | default | purpose |
|---|---|---|---|
| `LOG_LEVEL` | no | `info` | log level |
| `LOG_VERBOSE` | no | `false` | verbose logs |
| `OTEL_ENABLED` | no | `false` | enables otel exporter |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | no | empty | otlp endpoint |
| `OTEL_SERVICE_NAME` | no | `visor` | service name |
| `OTEL_ENVIRONMENT` | no | `dev` | environment tag |
| `OTEL_INSECURE` | no | `false` | insecure otlp transport |

## self-evolution pipeline

| variable | required | default | purpose |
|---|---|---|---|
| `SELF_EVOLUTION_ENABLED` | no | `false` | enables self-evolution manager |
| `SELF_EVOLUTION_REPO_DIR` | no | `.` | repo root used by self-evolution commands |
| `SELF_EVOLUTION_PUSH` | no | `false` | allows push after commits |

## level-up env file

`levelups` usually rely on `.levelup.env` values (template: `.levelup.env.example`) for sidecar-specific settings like imap/smtp credentials, cloudflared token, and obsidian container paths.
