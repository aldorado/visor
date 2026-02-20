# Level-up manifest (`levelup.toml`)

Defines one sidecar capability pack that visor can enable/disable.

## Required fields

```toml
name = "email-himalaya"
display_name = "Himalaya Email"
version = "0.1.0"
description = "Email send/receive sidecar via Himalaya"
compose_file = "docker-compose.levelup.email-himalaya.yml"
healthcheck = "http://127.0.0.1:8091/health"
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

## Optional fields

```toml
kind = "infra"            # default: "infra"
enabled_by_default = false # default: false
tags = ["email", "imap", "smtp"]
proxy_service = "obsidian" # optional: service name for proxy route generation
proxy_port = 3000           # optional: container port for proxy route generation
```

## Runtime rules

- compose assembly order is deterministic: base compose file first, then overlays in selected order.
- overlay file paths are resolved relative to the base compose file directory.
- every entry in `required_env` must exist after env layering (`.env` + `.levelup.env` + process env). missing keys are hard errors.
- before any apply/up/down operation, merged config must pass `docker compose ... config` validation.
- `healthcheck` is a simple probe target for status reporting.
- manifests are discovered under `levelups/*/levelup.toml`.
- visor itself is never declared in these compose files (sidecars only).
