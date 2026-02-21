# visor coordination

lightweight handoff board for humans + agents.

## source of truth

- scope/roadmap: `visor.forge.md`
- execution ownership + handoff: this file

if they conflict: `visor.forge.md` wins.

## status snapshot

done milestones:
`M0, M0b, M1, M2, M4, M5, M6, M7, M8, M8a, M10, M11, M12`

open milestones:
- `M3` remote memory sync tail
- `M9` multi-subagent orchestration

## workflow (mandatory)

1) read `visor.forge.md` + this file before coding
2) add claim under *active claims*
3) execute one iteration/chunk at a time
4) add handoff note (what changed, tests, commit)
5) user report format: `milestone/iteration -> short outcome -> commit`

## active claims

- [claim] none

## handoff log

### 2026-02-20
- `M10 / it1` proxy base level-up + isolated networks — `bfd4ee7`
- `M10 / it2` dynamic subdomain routing + lifecycle tests — `a7cbf3b`
- `M10 / it3` auth/allow/deny + admin dashboard route — `27ca2a6`
- `M11 / it1` forgejo level-up + bootstrap + proxy wiring — `79b926e`
- `M11 / it2` auto-push integration + git_push control — `954bb1b`
- `M11 / it3` forgejo webhook + repo skill + readme auto-gen — `370e09d`
- `M12 / it1` first-run detection + setup actions core — `ffe8d38`
- `M12 / it2` level-up setup actions + forgejo remote sync hook — `0b31da0`
- `M12 / it3` setup finish flow (personality/test/summary/cleanup) — `64d409a`
- `M12 / research+hardening` setup strategy + validate_openai + recommended preset — `912a104`, `2d7a51b`, `38fcec9`, `4142a28`, `71e1781`
