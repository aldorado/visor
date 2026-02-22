# visor agents

single workflow for all agents:
- strategy + scope live in `visor.forge.md`
- execution + handoff live in `COORDINATION.md`
- no backlog.md layer

## required preflight (always)
1) read `visor.forge.md` to pick milestone/iteration
2) read `COORDINATION.md` to check active claims + latest handoff
3) write your claim in `COORDINATION.md` before code changes

claim format:
`[claim] <agent> | <milestone/iteration> | <branch> | <time>`

## required handoff (always)
append to `COORDINATION.md`:
- milestone + iteration completed
- files changed
- tests run
- risks / follow-up
- commit hash

## user reporting rule (anuar)
- always report `milestone + iteration` first
- then short result + commit hash
- no task-id-only reports

## execution boundary
- visor runs host-native only (never inside docker compose)
- docker compose is sidecars/level-ups only

## prompt/skill metadata sync
- canonical source is `skills/` + `.pi/SYSTEM.md`
- mirrors must stay in sync: `.claude/`, `.gemini/`, `.agents/`
