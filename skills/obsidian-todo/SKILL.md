---
name: obsidian-todo
description: Internal skill for adding todos to Obsidian. Triggers when a voice or text message starts with "todo", "to do", or "to-do".
user-invocable: false
---

# Obsidian Todo

When a message starts with todo/to do/to-do, strip that prefix and append a cleaned task to today's todo file.

## Preconditions

1. Read `OBSIDIAN_VAULT_PATH` from environment.
2. If missing or path does not exist, stop and tell the user Obsidian is not configured.

## Location

`$OBSIDIAN_VAULT_PATH/todos/YYYY-MM-DD.md` (Vienna timezone)

## Format

```markdown
# Todos 2026-02-17

- [ ] Sync Registry Worker: Expand korrekt machen
- [ ] Search: eigene Collection für Content und Tag Score aufsetzen
```

## Rules

- Create the file if it doesn't exist, with header `# Todos YYYY-MM-DD`.
- Append a new `- [ ] ` line.
- Summarize/clean up dictated text into a clear actionable task.
- Keep language as spoken (de/en).
- Confirm shortly ("added").
