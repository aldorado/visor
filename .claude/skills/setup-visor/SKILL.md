---
name: setup-visor
description: "Use when the user wants guided first-run setup for visor, like 'setup visor', 'starte visor setup', 'führ mich durchs setup', or 'ich hab pi gestartet, was jetzt'."
user-invocable: true
argument-hint: "[project-folder optional]"
---

# Setup Visor

Guide the *user* through visor setup step-by-step. This skill is for onboarding/setup, not forge execution.

## Intent

- do not switch into project implementation mode
- do not create execution boards or iteration commits
- keep the flow user-guided and checkpointed

## Setup flow (M12-aligned)

0. Prerequisites check + install (before ingress/env)
   - verify required tools are installed and runnable
   - minimum checks: `go version`, `git --version`, `docker --version`, `docker compose version`, `curl --version`
   - if using direct dns + caddy: also verify caddy is installed
   - if something is missing: install it immediately (guided + explicit commands), then re-check versions
   - ask once before running privileged install commands (`sudo`)

1. Ingress + domain routing (before env)
   - decide mode with user: `cloudflare tunnel` or `direct dns + caddy`
   - pick one base public url for webhook (e.g. `https://bot.example.com`)
   - webhook target must be `<base-url>/webhook` (not root `/`)
   - define subdomain plan for level-ups (e.g. `forgejo.`, `obsidian.`, etc.)
   - verify dns/tunnel is pointing correctly before webhook setup
   - for direct dns + caddy with level-up proxy: keep host caddy on `:80/:443`, run level-up proxy on localhost high ports (e.g. `127.0.0.1:18080/18443`) and route subdomains via host caddy
   - when using routes like `<sub>.visor.<domain>`, set `PROXY_DOMAIN=<domain>` (not `visor.<domain>`)
   - only then continue to env + token steps

2. Core setup
   - first check if `.env` already exists
   - if `.env` exists: do *not* overwrite/replace it from template
   - if `.env` is missing: create it from `.env.example`
   - only patch keys via `env_set` / `env_unset` (no full-file rewrite from model output)
   - collect required values (`TELEGRAM_BOT_TOKEN`, `USER_PHONE_NUMBER`)
   - run telegram validation
   - optionally run openai validation
   - set webhook url/secret (`webhook_url` should include `/webhook`)
   - run `/health` check

3. Optional level-ups
   - ask user to choose: `none`, `recommended`, or explicit list
   - collect needed `.levelup.env` values
   - enable selected level-ups
   - validate level-ups
   - start level-ups
   - run level-up health check

4. Finish
   - ask personality choice (keep/custom)
   - optionally send a test message
   - write setup summary
   - cleanup setup hints

## How to execute changes

When applying setup changes, use exactly one setup action json block in final response:

```json
{
  "setup_actions": {
    "env_set": {},
    "env_unset": [],
    "validate_telegram": false,
    "validate_openai": false,
    "webhook_url": "",
    "webhook_secret": "",
    "check_health": false,
    "levelup_env_set": {},
    "levelup_env_unset": [],
    "levelup_preset": "",
    "enable_levelups": [],
    "disable_levelups": [],
    "validate_levelups": false,
    "start_levelups": false,
    "check_levelups": false,
    "sync_forgejo_remote": false,
    "personality_choice": "",
    "personality_file": "",
    "personality_text": "",
    "send_test_message": "",
    "write_summary": false,
    "cleanup_setup_hints": false
  }
}
```

## systemd / reboot persistence

Setup should also ensure process-manager persistence:
- verify `visor.service` exists and has `Restart=always`
- ensure service is enabled (`systemctl enable visor`)
- if not possible due to permissions, ask the user for one exact sudo command

## Rules

- one step at a time, ask before applying impactful changes
- prerequisites check/install comes first, then ingress decision (cloudflare tunnel vs direct dns+caddy)
- for missing deps: propose exact install command and run it when user confirms
- if sudo is unavailable, stop and give one copy/paste command block for user
- never replace an existing `.env` with `.env.example`
- if `.env` exists, preserve existing keys and only patch requested values
- keep messages short and practical
- never expose level-up proxy ports publicly when host ingress already exists; prefer localhost bind + host reverse-proxy
- if something fails, report exact command/output and next fix step
- work only inside `/root/code/<project-folder>/`

## Quick invoke examples

- `/setup-visor visor`
- `/setup-visor`
- "führ mich durchs visor setup"
