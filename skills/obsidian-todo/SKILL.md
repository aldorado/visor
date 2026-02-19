---
name: obsidian-todo
description: Internal skill for adding todos to obsidian. Triggers when a voice or text message starts with "todo" or "to do" or "to-do".
user-invocable: false
---

# Obsidian Todo

When a voice or text message starts with "todo", "to do", or "to-do" (case-insensitive), strip that prefix and add a cleaned-up todo item to today's todo file.

## Location

`~/obsidian/sibwax/todos/YYYY-MM-DD.md` (Vienna timezone for date)

## Format

```markdown
# Todos 2026-02-17

- [ ] Sync Registry Worker: Expand korrekt machen
- [ ] Search: eigene Collection für Content und Tag Score aufsetzen
```

## Rules

- Create the file if it doesn't exist yet, with the `# Todos YYYY-MM-DD` header
- Append a new `- [ ] ` line with the todo
- IMPORTANT: Summarize and clean up the dictated text — the user may stutter, repeat themselves, or ramble. Extract the actual actionable task and write it concisely
- Keep the language the user used (German stays German, English stays English)
- Don't confirm with a long message, just a short "added" or similar
