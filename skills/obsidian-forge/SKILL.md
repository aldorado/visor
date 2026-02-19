---
name: obsidian-forge
description: Internal skill for promoting mature ideas into execution-ready forge plans with milestones, iterations, research tasks, and prompt-ready execution chunks.
user-invocable: false
---

# Obsidian Forge Planner

When an idea is mature and close to execution, promote it from `ideas/` to `forge/`.

## Trigger conditions

Use this skill when the user asks for:
- a concrete implementation plan
- something "ready to execute"
- milestones / iterations / research tasks
- moving an idea from "ideas" to "forge"

## Locations

- ideas: `~/obsidian/sibwax/ideas/`
- forge: `~/obsidian/sibwax/forge/`

## Promotion rules

Only promote if these are true:
1. scope is clear enough to build now
2. outcome is measurable (definition of done)
3. next 1-3 implementation prompts can be written concretely

If not ready, keep it in `ideas/` and ask focused clarification questions.

## Required forge format

Every forge plan must include:

```markdown
# Plan Title

## Vision

## Scope (v1)

## Success metrics

## Milestones
### M1 ...
#### Iteration 1 ...
- [ ] ...

## Research tasks
- [ ] ...

## Prompt-ready execution chunks
### Prompt 1
"..."

## Definition of Done

#forge #...

> promoted from [ideas/original-file](../ideas/original-file.md)
```

## Steps

1. Read the source idea file in `ideas/`.
2. Read relevant existing forge examples for style consistency (e.g. `forge/visor.md`).
3. Create a new file in `forge/` with the same slug/theme as the idea.
4. Write concrete milestones + iterations + research tasks.
5. Add at least 2-3 prompt-ready execution chunks.
6. Update the idea file with a `promoted to forge` backlink.
7. Confirm briefly to the user.

## Writing style

- concrete, buildable, no vague strategy fluff
- action-first wording
- short enough to execute, detailed enough to prompt from directly
- bias toward simple implementation over architecture theater
