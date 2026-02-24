---
name: obsidian-log
description: Internal skill for logging voice entries to Obsidian. Triggers when a voice message starts with "log entry" or "logentry".
user-invocable: false
---

# Obsidian Log

When a voice message starts with log entry/logentry, strip the prefix and append to today's log file.

## Preconditions

1. Use `/root/obsidian/Sibwax` as the vault root.
2. If path does not exist, stop and tell the user Obsidian is not configured.

## Location

`/root/obsidian/Sibwax/logs/YYYY-MM-DD.md`

## Format

```markdown
# Log - 2026-02-13

## 10:41
went to the gym, did legs today

## 14:30
had a meeting about the new project
```

## Rules

- Create daily file with header `# Log - YYYY-MM-DD` if missing.
- Append entries with `## HH:MM` (Vienna time).
- Keep wording natural; only light cleanup.
- Confirm shortly ("logged").
