---
name: forge-blueprint-start
description: Use when the user says the forge blueprint for a project is finished, like "forge blueprint ist fertig" or "blueprint ist fertig" and wants project kickoff scaffolding.
user-invocable: false
---

# Forge Blueprint Kickoff

When the user says a forge blueprint is finished, execute this exact flow.

## Required flow

1. Ask first for:
   - project name
   - folder name under `/root/code/`

   Also suggest 2-4 clean folder-name options.

2. Create project folder under `/root/code/<folder-name>`.

3. *Mandatory:* copy both docs into the new folder (never forge-only):
   - source idea file from `~/obsidian/sibwax/ideas/`
   - source forge file from `~/obsidian/sibwax/forge/`

4. In the new folder, name files like:
   - `project-name.md` (idea file)
   - `project-name.forge.md` (forge file)

   Rule: forge document must include `.forge.` before `.md`.

5. Initialize git in the new folder:
   - `git init`

6. Send a short completion signal to the user when done.

## Notes

- If project/folder naming is ambiguous, stop and ask before creating files.
- If source docs are missing, fail fast and tell the user exactly which path is missing.
- Keep it simple: no extra setup unless user asks.
