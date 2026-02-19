---
name: chat-history
description: Internal skill to retrieve recent chat history. Use when you need context from previous conversations - when the user references something you discussed before, or when "last time" or "earlier" is mentioned.
user-invocable: false
---

# Chat History Retrieval

Retrieve recent messages from our chat history across all sessions.

## Usage

Run this bash command to get the last N session files (most recent messages):

```bash
ls -t data/sessions/*.md 2>/dev/null | head -5 | while read f; do echo "=== $(basename "$f") ==="; cat "$f"; echo; done
```

Change `head -5` to get more or fewer session files.

For a quick look at just the most recent session:

```bash
ls -t data/sessions/*.md 2>/dev/null | head -1 | xargs cat
```

To search for a specific topic across all sessions:

```bash
grep -ril "search term" data/sessions/ | head -5 | while read f; do echo "=== $(basename "$f") ==="; cat "$f"; echo; done
```

## When to Use

- User says "remember when we talked about X" but you don't have it in context
- User references "last time" or "earlier today" or "yesterday"
- User asks "what did I say about X"
- You need to check what was discussed recently

## Output Format

Session files are markdown with timestamped messages:

```
[2026-01-31 00:57]
user: Hey you there
ubik: hey, what's up?

---

[2026-01-30 22:15]
user: ...
ubik: ...
```

## Notes

- All sessions are stored permanently in data/sessions/ as markdown files
- Files are named with timestamps, so `ls -t` gives newest first
- Messages include timestamps for context
- Search through the output for relevant keywords
