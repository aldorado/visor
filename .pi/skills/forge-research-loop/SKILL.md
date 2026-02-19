---
name: forge-research-loop
description: Use after forge bootstrap when the user wants research tasks from a project .forge.md executed one by one, with findings written back into project forge and idea files.
user-invocable: false
---

# Forge Research Loop

Use this after project bootstrap is done.

## Trigger context

Typical user phrases:
- "start research from the forge file"
- "mach die research steps"
- "geh die forge tasks einzeln durch"
- "nach forge-bootstrap weitermachen"

## Required flow

1. Ask for project folder if not already clear (`/root/code/<project-folder>`).
2. Open the project `*.forge.md` and identify research tasks/checklist items.
3. Execute exactly one research step at a time.
4. After each step:
   - write the finding into the project forge file (`*.forge.md`)
   - update the project idea file (`*.md`) if the finding changes scope/decisions
   - send a short report to the user
   - explicitly ask: continue with the next research step?
5. Wait for user confirmation before running the next step.
6. When all research steps are done:
   - create `/root/code/<project-folder>/agents.md`
   - include concise agent roles/work split inferred from the finished research and forge plan
   - send final done signal.

## File rules

- Never edit files in obsidian during this phase; only files inside the project folder under `/root/code/<project-folder>`.
- Keep forge as the execution truth and idea as high-level intent; update both when needed.
- Fail fast if required files are missing and tell the user the exact missing path.

## Reporting style

- short
- concrete
- no implementation until user says continue
