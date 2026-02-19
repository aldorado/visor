# build a level-up (template)

this guide defines the minimal pattern to add a new visor level-up.

## 1) add a manifest

create: `levelups/<name>/levelup.toml`

example:

```toml
name = "email-himalaya"
display_name = "Himalaya Email"
version = "0.1.0"
description = "Email send/receive sidecar via Himalaya"
kind = "infra"
enabled_by_default = false
compose_file = "docker-compose.levelup.email-himalaya.yml"
healthcheck = "http://127.0.0.1:8091/health"
tags = ["email", "imap", "smtp"]
required_env = [
  "HIMALAYA_IMAP_HOST",
  "HIMALAYA_IMAP_PORT",
  "HIMALAYA_SMTP_HOST",
  "HIMALAYA_SMTP_PORT",
  "HIMALAYA_EMAIL",
  "HIMALAYA_PASSWORD",
  "HIMALAYA_USE_TLS",
]
```

rules:
- `name` must be unique
- `compose_file` must exist
- every key in `required_env` must be present after env layering (`.env` + `.levelup.env` + process env)

## 2) add compose overlay

create: `docker-compose.levelup.<name>.yml`

rules:
- this file must contain only sidecars, never visor itself
- paths should be host bind mounts when visor needs direct file access
- keep restart policy explicit (`unless-stopped`)

## 3) add env contract

add required keys to `.levelup.env.example`.

rules:
- no secrets in compose file
- use env vars only

## 4) validate

run:

```bash
go run ./cmd/visor-admin --project-root . levelup list
go run ./cmd/visor-admin --project-root . levelup enable <name>
go run ./cmd/visor-admin --project-root . levelup validate
```

if validation fails, do not apply.

## 5) runtime bridge (if capability needs it)

for sidecars that expose capability (email, storage, etc.), add a bridge in `internal/levelup/<capability>/`.

himalaya reference pieces:
- poller (`poller.go`) for inbound flow
- action parser (`actions.go`) for agent output -> capability action
- client (`himalaya.go`) for external command integration

## 6) tests

minimum:
- unit tests for parser/bridge code
- one integration-ish roundtrip test for the capability flow
- mount/reachability smoke tests if sidecar has volumes or http endpoint

## reference level-ups in this repo

base connectivity reference:
- `levelups/cloudflared/levelup.toml`
- `docker-compose.levelup.cloudflared.yml`

minimal generic example:
- `levelups/echo-stub/levelup.toml`
- `docker-compose.levelup.echo-stub.yml`
