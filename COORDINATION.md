# visor coordination board (backup)

backup coordination layer for humans + multi-agents.

## source of truth and execution flow

- strategic source of truth: `visor.forge.md` (milestones, iterations, scope intent)
- execution source of truth: `backlog/` (backlog.md tasks and statuses)
- this file: backup handoff + operating rules

if views diverge:
1) `visor.forge.md` decides strategy/scope
2) `backlog/` decides task execution state
3) this file gets updated after merges/handoffs

## status snapshot (2026-02-20)

### done milestones
- `M0` host-native + level-up foundation
- `M0b` observability baseline
- `M1` skeleton
- `M2` agent process manager
- `M4` voice pipeline
- `M6` skills system
- `M7` multi-backend + auto-switch
- `M8` self-evolution
- `M8a` release hardening + repo polish

### partially open milestones
- `M3` memory: 2 open items (remote sync protocol + incremental sync)

### not started / mostly open
- `M9` optional multi-subagent orchestration (18 open)
- `M10` reverse proxy level-up (13 open)
- `M11` forgejo level-up (12 open)
- `M12` interactive first-run onboarding (18 open)

## backlog.md is now central for execution

migration status:
- all current open `visor.forge.md` checkboxes are mirrored into `backlog/tasks`
- total open forge items mirrored: 63

usage rule:
- new implementation work must be created/updated in backlog.md first
- after merge, reflect completion back into `visor.forge.md` checkboxes

quick commands:
- `npx backlog.md task list --plain`
- `npx backlog.md board`
- `npx backlog.md task view TASK-<n>`

## now / next recommendation

1. execute M3 remaining work first (`TASK-1` then `TASK-2`)
2. then pick one major stream (`M10` infra exposure or `M12` onboarding)
3. keep `M9` optional until core infra/onboarding streams are stable

## multi-agent coordination rules

### task claiming
before changing code, claim exactly one task block:

```text
[claim] <agent-name> | <milestone/iteration/task> | <branch> | <time>
```

append claim + release notes to this file under `work log`.

### branch ownership
- one task block per branch
- no mixed milestones in one branch
- merge fast after green tests to avoid drift

### safe handoff
each agent handoff must include:
- files changed
- tests run
- risks / follow-up
- exact commit hash

### user communication contract (anuar)
when reporting progress to the user:
- always include `milestone + iteration` first (example: `M10 / Iteration 1`)
- backlog task id is optional and secondary (can be appended)
- avoid reporting only `TASK-xx` without milestone/iteration context

## work log

### 2026-02-20
- completed `M8a it1` + `M8a it2`
- repo/docs polish merged (`.gitignore`, `LICENSE`, `CONTRIBUTING`, `README`, `CHANGELOG`, config/ops docs)
- completed `M1` final e2e closure via webhook→agent→telegram delivery test harness
- completed `M5 it2.5` quick actions (`done/snooze/reschedule`), natural time parsing, idempotency guard, and tests
- initialized backlog.md and migrated all currently open forge checkboxes into backlog tasks (63 mirrored items)
- set communication rule: report to user with milestone/iteration first, backlog task id optional
- latest related commits: `bbd6d1b`, `7970e20`, `5c51feb`, `7eabfde`
