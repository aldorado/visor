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

1. Core setup
   - first check if `.env` already exists
   - if `.env` exists: do *not* overwrite/replace it from template
   - if `.env` is missing: create it from `.env.example`
   - only patch keys via `env_set` / `env_unset` (no full-file rewrite from model output)
   - collect required values (`TELEGRAM_BOT_TOKEN`, `USER_PHONE_NUMBER`)
   - run telegram validation
   - optionally run openai validation
   - set webhook url/secret
   - run `/health` check

2. Optional level-ups
   - ask user to choose: `none`, `recommended`, or explicit list
   - collect needed `.levelup.env` values
   - enable selected level-ups
   - validate level-ups
   - start level-ups
   - run level-up health check

3. Finish
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
- never replace an existing `.env` with `.env.example`
- if `.env` exists, preserve existing keys and only patch requested values
- keep messages short and practical
- if something fails, report exact command/output and next fix step
- work only inside `/root/code/<project-folder>/`

## Quick invoke examples

- `/setup-visor visor`
- `/setup-visor`
- "führ mich durchs visor setup"
