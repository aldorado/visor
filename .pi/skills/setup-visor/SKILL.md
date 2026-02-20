---
name: setup-visor
description: "Use when the user wants a one-shot project kickoff in visor, like 'setup visor', 'starte visor setup', 'kickoff forge execution', or 'was soll ich nach pi eingeben'."
user-invocable: true
argument-hint: "[project-folder optional]"
---

# Setup Visor

Kick off a project execution flow in one command so the user does not need a long manual prompt.

## What this skill does

1. Resolve project folder.
   - If user gave one, use it.
   - Else default to current folder.
   - If unclear, ask one short question.

2. Read context files before coding:
   - `README.md` (if present)
   - `COORDINATION.md` (if present)
   - `*.forge.md` (required)
   - matching idea file `*.md` (if present)

3. Initialize execution board in `README.md`:
   - current milestone + iteration focus
   - granular executable TODOs
   - parallel worksplit
   - file-touch map to avoid merge collisions

4. Ensure process supervisor setup (systemd):
   - check whether `visor.service` exists and is enabled
   - if missing, create/update unit using repo docs (`docs/ubuntu-24-noob-install*.md`)
   - service must load `.env`, use project root as `WorkingDirectory`, and set `Restart=always`
   - run `sudo systemctl daemon-reload`
   - run `sudo systemctl enable --now visor`
   - verify with `systemctl status visor --no-pager` and `curl -s http://localhost:8080/health`

5. Start execution mode:
   - execute exactly one iteration chunk
   - run tests/lint for touched scope
   - update `README.md` + forge progress
   - commit iteration changes with clear message
   - send short report + ask: "nächste iteration?"

## Rules

- no multi-iteration silent batching
- stop on ambiguity and ask one precise question
- fail fast on missing required files with exact path
- if systemd permissions are missing, stop and ask user to run the exact `sudo` command
- work only inside `/root/code/<project-folder>/`
- keep output short and checkpoint-driven

## Quick invoke examples

- `/setup-visor visor`
- `/setup-visor`
- "mach setup visor im visor repo"
