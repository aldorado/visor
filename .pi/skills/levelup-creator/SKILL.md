---
name: levelup-creator
description: Use when asked to create a new visor level-up, add infrastructure sidecars, or extend visor with new levelup capabilities.
user-invocable: false
---

# Levelup Creator

Create a new visor level-up end-to-end using the M0 docs and validation flow.

## Source of truth

Always read these first:
- `docs/levelup-authoring.md`
- `docs/levelup-failure-modes.md`
- `docs/levelup-manifest.md`

## Required output for every new level-up

1) manifest
- create `levelups/<name>/levelup.toml`
- include: `name`, `compose_file`, `required_env`, `healthcheck`

2) compose overlay
- create `docker-compose.levelup.<name>.yml`
- sidecars only (never visor service)

3) env contract
- add all required keys to `.levelup.env.example`
- no secrets in compose yaml

4) optional runtime bridge
- if level-up exposes capabilities (mail, storage, search, etc.), add runtime bridge code in `internal/levelup/<capability>/`
- add focused tests for parser/bridge logic

5) validation
- run:
  - `go run ./cmd/visor-admin --project-root . levelup list`
  - `go run ./cmd/visor-admin --project-root . levelup enable <name>`
  - `go run ./cmd/visor-admin --project-root . levelup validate`
- fail fast if validation fails and report exact reason

6) docs sync
- update `README.md` execution board for touched iteration
- update `visor.forge.md` checklist/progress
- update `visor.md` if architecture intent changed

7) commit
- commit with a clear level-up scoped message

## Guardrails

- keep visor host-native; do not put visor itself into compose
- prefer simple, deterministic files over abstractions
- if requirements are ambiguous, ask one precise question and wait
