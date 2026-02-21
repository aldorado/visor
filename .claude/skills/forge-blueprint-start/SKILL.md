---
name: forge-blueprint-start
description: Use when the user says the forge blueprint for a project is finished and wants kickoff scaffolding.
user-invocable: false
---

# Forge Blueprint Kickoff

When user says blueprint is done, run this flow.

## Required flow

1. Ask for:
   - project name
   - target folder name under the projects root
   - source idea/forge slug (if unclear)
   Suggest 2-4 clean folder-name options.

2. Resolve projects root:
   - use `PROJECTS_ROOT` env if set
   - otherwise ask user for absolute path

3. Create `<projects-root>/<folder-name>`.

4. *Mandatory:* copy both source docs into the new folder (never forge-only):
   - idea source: `<obsidian-vault>/ideas/<slug>.md`
   - forge source: `<obsidian-vault>/forge/<slug>.md`

5. Name files in target folder:
   - `project-name.md`
   - `project-name.forge.md`

6. Initialize git in target folder (`git init`).

7. Send short done signal.

## Obsidian source path

Read `OBSIDIAN_VAULT_PATH` from `.levelup.env`.
If missing or path does not exist, stop and tell the user Obsidian level-up is not configured/enabled.

## Notes

- if naming is ambiguous, stop and ask.
- fail fast on missing source files with exact path.
- no extra setup unless user asks.
