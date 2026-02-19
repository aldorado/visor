---
name: forge-execution
description: Use when the user wants to start implementing a project from its forge blueprint, like "start forge execution", "arbeite das forge blueprint ab", "starte mit milestone 1", or "setz das forge jetzt um".
user-invocable: false
---

# Forge Execution

Execute a project from `*.forge.md` step by step, with tight checkpoints.

## Trigger context

Typical user phrases:
- "start forge execution"
- "setz das forge um"
- "arbeite milestone für milestone ab"
- "mach iteration für iteration"
- "fang mit der umsetzung an"

## Required flow

1. Ask for project folder if unclear (`/root/code/<project-folder>`).
2. Read both project docs:
   - `*.forge.md` (execution source of truth)
   - `*.md` (idea intent)
3. Before coding anything, create or refresh `/root/code/<project-folder>/README.md` with:
   - current milestone + iteration focus
   - granular TODOs as small executable tasks
   - clear ownership/work-split blocks so multiple agents can work in parallel without collisions
   - file-level touch map (which files each task/agent should touch)
4. Execute exactly one iteration chunk at a time (or one milestone chunk if no iterations are defined).
5. After each chunk:
   - update `README.md` task state (todo/in-progress/done)
   - update `*.forge.md` progress/checklists and decisions
   - update `*.md` if scope or direction changed
   - commit the iteration changes in git before moving on
   - report what was done, what changed, and what is blocked
   - ask explicit checkpoint question before continuing
6. Stop whenever a decision is ambiguous or missing input is required. Ask a precise question and wait.
7. Repeat until milestone is complete, then ask to proceed to next milestone.

## Execution rules

- one chunk at a time. no silent batch execution across multiple milestones.
- keep changes minimal and concrete.
- fail fast if required files are missing; report exact missing path.
- do not edit obsidian source files during execution. only work inside `/root/code/<project-folder>/`.
- README must stay current. it is the shared coordination layer for parallel agents.
- each finished iteration chunk must end with a git commit in the project repo (clear message, iteration-scoped).

## Reporting style

- short
- concrete
- checkpoint-driven
