---
name: forge-execution
description: Use when the user wants to implement a project from its forge blueprint milestone-by-milestone.
user-invocable: false
---

# Forge Execution

Execute a project from `*.forge.md` step by step with strict checkpoints.

## Required flow

1. Ask for project folder path if unclear (absolute path, no assumptions).
2. Read both project docs:
   - `*.forge.md` (execution truth)
   - `*.md` (idea intent)
3. Before coding, create/refresh `<project-folder>/README.md` with:
   - current milestone + iteration focus
   - granular TODOs
   - parallel work split
   - file touch map
4. Execute exactly one iteration chunk at a time.
5. After each chunk:
   - update `README.md`
   - update `*.forge.md`
   - update `*.md` if scope changed
   - commit before continuing
   - report outcome/changes/blockers
   - ask explicit checkpoint question
6. Stop on ambiguity and ask precise question.

## Execution rules

- one chunk at a time, no silent batch across milestones
- minimal concrete changes
- fail fast if files are missing
- only edit files inside the chosen project folder
- README stays current as coordination layer
- each finished chunk ends with a git commit

## Reporting style

short, concrete, checkpoint-driven.
