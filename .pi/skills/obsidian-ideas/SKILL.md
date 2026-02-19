---
name: obsidian-ideas
description: Internal skill for saving ideas to obsidian. Triggers when user dictates an idea or asks to save something to ideas.
user-invocable: false
---

# Obsidian Ideas

When the user shares an idea (voice or text) and wants it saved, structure it and write it to the ideas folder.

## Location

`~/obsidian/sibwax/ideas/`

One markdown file per idea, named with a short slug: `idea-name.md`

## Format

Don't just copy the user's raw text. Structure it:

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

- Structure and clean up — don't just dump raw transcription
- Use obsidian tags at the bottom where relevant (#friday #feature #project etc)
- If the idea relates to an existing file, update it instead of creating a new one
- Confirm briefly when saved
