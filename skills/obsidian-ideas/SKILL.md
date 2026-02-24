---
name: obsidian-ideas
description: Internal skill for saving ideas to Obsidian. Triggers when user dictates an idea or asks to save something to ideas.
user-invocable: false
---

# Obsidian Ideas

Save user ideas into the Obsidian vault.

## Preconditions

1. Use `/root/obsidian/Sibwax` as the vault root.
2. If path does not exist, stop and tell the user Obsidian is not configured.

## Location

`/root/obsidian/Sibwax/ideas/`

One markdown file per idea, named with a short slug: `idea-name.md`.

## Format

```markdown
# Idea Title

## Core idea
Brief summary of what the idea is about.

## Details
- Key points extracted from what the user said
- Structured and cleaned up

## Open questions
- Things that still need to be figured out
```

## Rules

- Structure and clean up — don't just dump raw transcription.
- Use Obsidian tags at the bottom where relevant.
- If the idea relates to an existing file, update it instead of creating a new one.
- Confirm briefly when saved.
