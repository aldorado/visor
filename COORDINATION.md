# visor coordination board

single coordination board for humans + multi-agents.

## source of truth

- detailed planning and task definitions: `visor.forge.md`
- this file is the execution snapshot for collaboration (what is open, what is next, who is working on what)
- `backlog/` (backlog.md tool) is the active execution layer for currently claimed implementation tasks

if these views diverge:
1) `visor.forge.md` wins for milestone intent
2) `backlog/` wins for in-flight task state
3) update this file right after merge/iteration

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

## backlog.md pilot (active)

scope rule:
- use backlog.md only for active execution tasks (not full roadmap mirroring)
- keep `visor.forge.md` as strategic milestone spec

active pilot queue:
- `TASK-1` — M3 sync protocol design
- `TASK-2` — M3 incremental sync (depends on TASK-1)

quick commands:
- `npx backlog.md task list`
- `npx backlog.md board`

## now / next recommendation

1. execute `TASK-1` then `TASK-2` (close remaining M3 open items)
2. then choose one major stream: `M10` (infra exposure) or `M12` (onboarding)
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

## work log

### 2026-02-20
- completed `M8a it1` + `M8a it2`
- repo/docs polish merged (`.gitignore`, `LICENSE`, `CONTRIBUTING`, `README`, `CHANGELOG`, config/ops docs)
- completed `M1` final e2e closure via webhook→agent→telegram delivery test harness
- completed `M5 it2.5` quick actions (`done/snooze/reschedule`), natural time parsing, idempotency guard, and tests
- initialized backlog.md pilot (`backlog/config.yml`) and created active M3 execution tasks (`TASK-1`, `TASK-2`)
- latest related commits: `bbd6d1b`, `7970e20`, `5c51feb`
