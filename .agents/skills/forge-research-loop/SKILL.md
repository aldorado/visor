---
name: forge-research-loop
description: Use after forge bootstrap when user wants research tasks from a project .forge.md executed one by one.
user-invocable: false
---

# Forge Research Loop

Run research tasks from a project forge file one step at a time.

## Required flow

1. Ask for project folder path if unclear (absolute path).
2. Open project `*.forge.md` and list research items.
3. Execute exactly one research step.
4. After each step:
   - write findings into `*.forge.md`
   - update project `*.md` if scope/decisions changed
   - send short report
   - ask: continue with next step?
5. Wait for user confirmation.
6. After all research is done:
   - create `<project-folder>/agents.md`
   - add concise roles/work split based on findings
   - send done signal.

## File rules

- only edit files inside the chosen project folder
- keep forge as execution truth and idea as intent
- fail fast on missing files and report exact path

## Reporting style

short, concrete, no implementation until user says continue.
