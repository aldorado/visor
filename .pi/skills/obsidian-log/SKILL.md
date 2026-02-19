---
name: obsidian-log
description: Internal skill for logging voice entries to obsidian. Triggers when a voice message starts with "log entry" or "logentry".
user-invocable: false
---

# Obsidian Log

When a voice message starts with "log entry" or "logentry" (case-insensitive), strip that prefix and append the transcribed text to today's log file.

## Log location

`~/obsidian/sibwax/logs/YYYY-MM-DD.md`

## Format

Each day gets one file. Entries stack chronologically:

```markdown
# Log - 2026-02-13

## 10:41
went to the gym, did legs today

## 14:30
had a meeting about the new project
```

## Rules

- Create the file if it doesn't exist yet, with the `# Log - YYYY-MM-DD` header
- Append new entries with `## HH:MM` (Vienna time) as subheader
- Keep the user's words natural — light cleanup only (no restructuring)
- Don't confirm with a long message, just a short "logged" or similar
