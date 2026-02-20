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
- [claim] none

## handoff log
### 2026-02-20
- `M10 / Iteration 1` done — proxy base level-up + isolated networks + route autogen — commit `bfd4ee7`
- `M10 / Iteration 2` done — dynamic subdomain routing + lifecycle tests — commit `a7cbf3b`
- `M10 / Iteration 3` done — per-subdomain auth/allow/deny + admin dashboard route — commit `27ca2a6`
- workflow decision: remove backlog.md layer, use `visor.forge.md` + `COORDINATION.md` only
- owner decision: M11 assigned to friday, ubik executes M12 stream
- `M11 / Iteration 1` done — Forgejo level-up: compose file, levelup.toml (subdomain=git), CMD-override bootstrap (admin user + visor-push token → /data/visor-push.token), proxy network wired — commit `79b926e`
- `M11 / Iteration 2` done — auto-push integration: internal/forgejo pkg (ReadToken, SyncRemote, PushBackground), selfevolve hook, levelup enable/disable hook, git_push structured output field, FORGEJO_HOST_PORT localhost port binding — commit `954bb1b`
- `M11 / Iteration 3` done — visibility + collaboration: POST /forgejo/webhook (push/PR → Telegram notification), EnsureReadme (API-based README creation on enable), forgejo-repos skill (list repos/commits/issues via API) — commit `370e09d`

- `M12 / Iteration 1` done — first-run setup detection + setup action pipeline (.env update, telegram token validate, webhook set, health check) — commit `ffe8d38`
- `M12 / Iteration 2` done — setup actions now support level-up selection/env/start/validate/health + forgejo remote sync hook — commit `0b31da0`
- `M12 / Iteration 3` done — setup finish actions: personality override, telegram test message, setup summary writer, setup-hints cleanup — commit `64d409a`
- `M12 / Research 1` done — first-run detection strategy confirmed: runtime state + /health liveness; CLAUDE/.pi stay policy-only — commit `912a104`
- `M12 / Research 2` done — platform setup flow documented: Telegram-first path finalized, Signal scoped as future dedicated transport milestone — commit `2d7a51b`
- `M12 / Research 3` done — interactive env validation documented (telegram/webhook/health + levelup validate), identified OpenAI-key validation gap with concrete probe recommendation — commit `38fcec9`
- `M12 / Research 4` done — optional level-up selection UX documented (staged chooser + recommended shortcut + deterministic apply order) — commit TBD
