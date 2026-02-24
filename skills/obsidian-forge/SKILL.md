---
name: obsidian-forge
description: Internal skill for promoting mature ideas into execution-ready forge plans with milestones, iterations, and research tasks.
user-invocable: false
---

# Obsidian Forge Planner

Promote mature ideas from `ideas/` to `forge/` inside the Obsidian vault.

## Preconditions

1. Use `/root/obsidian/Sibwax` as the vault root.
2. If path does not exist, stop and tell the user Obsidian is not configured.

## Locations

- ideas: `/root/obsidian/Sibwax/ideas/`
- forge: `/root/obsidian/Sibwax/forge/`

## Promotion rules

Only promote when scope is clear, outcome measurable, and next prompts are concrete.
If not ready, keep it in ideas and ask focused questions.

## Required forge format

```markdown
# Plan Title

## Vision
## Scope (v1)
## Success metrics
## Milestones
## Research tasks
## Prompt-ready execution chunks
## Definition of Done

#forge #...
```

## Steps

1. Read source idea file in `ideas/`.
2. Read relevant forge examples for style consistency.
3. Create matching forge file in `forge/`.
4. Add concrete milestones, iterations, research tasks.
5. Add at least 2 prompt-ready execution chunks.
6. Update idea file with backlink to forge file.
7. Confirm briefly.
