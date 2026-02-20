# visor coordination board

single execution board for humans + agents.

## source of truth
- strategy/source of truth: `visor.forge.md`
- execution/source of truth: this file (`COORDINATION.md`)
- backlog.md is removed (was overhead)

if in conflict:
1) `visor.forge.md` decides scope/plan
2) `COORDINATION.md` decides active ownership + latest execution state

## current status snapshot (2026-02-20)

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
- `M10` reverse proxy level-up (it1-it3)

### open milestones
- `M3` memory sync tail (2 items, low prio)
- `M9` multi-subagent orchestration
- `M11` forgejo level-up
- `M12` interactive first-run onboarding

## agent workflow (mandatory)

### 1) preflight
before coding, read:
- `visor.forge.md` (target milestone/iteration)
- this file (active claims + latest handoff)

### 2) claim
append one claim line under `active claims`:
`[claim] <agent> | <milestone/iteration> | <branch> | <time>`

### 3) execute
- work one iteration at a time
- avoid mixing milestones in one branch

### 4) handoff
append under `handoff log`:
- milestone + iteration
- files changed
- tests run
- risks/follow-up
- commit hash

### 5) user reporting contract (anuar)
- always start with `milestone + iteration`
- include short outcome + commit hash
- keep it compact

## active claims
- [claim] ubik | M12/Iteration 1 | main | 2026-02-20T14:44

## handoff log
### 2026-02-20
- `M10 / Iteration 1` done — proxy base level-up + isolated networks + route autogen — commit `bfd4ee7`
- `M10 / Iteration 2` done — dynamic subdomain routing + lifecycle tests — commit `a7cbf3b`
- `M10 / Iteration 3` done — per-subdomain auth/allow/deny + admin dashboard route — commit `27ca2a6`
- workflow decision: remove backlog.md layer, use `visor.forge.md` + `COORDINATION.md` only
- owner decision: M11 assigned to friday, ubik executes M12 stream
- `M11 / Iteration 1` done — Forgejo level-up: compose file, levelup.toml (subdomain=git), CMD-override bootstrap (admin user + visor-push token → /data/visor-push.token), proxy network wired — commit `79b926e`
