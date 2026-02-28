---
name: memory-lookup
description: Internal skill to retrieve relevant memories for any topic. Use proactively when context about the user's preferences, projects, or past decisions could be helpful.
---

# Memory Lookup

You have a semantic memory system stored in `data/memories.parquet`. Use this skill to search your memories when you need context.

## When to use this

Use this proactively when:
- user mentions a topic you might have stored info about
- you need to recall preferences, past decisions, project details
- user references something from the past
- context would help you give a better response

## How to search

Use the canonical runtime command:

```bash
go run ./cmd/memorylookup -query "YOUR_QUERY_HERE"
```

or via script wrapper:

```bash
./scripts/memory-lookup.sh -query "YOUR_QUERY_HERE"
```

Replace `YOUR_QUERY_HERE` with your search query. Be specific - semantic lookup works better with concrete phrasing.

## Settings

Defaults: `-threshold 0.3`, `-max-results 5`, `-min-results 3`.

Examples:

```bash
go run ./cmd/memorylookup -query "query" -threshold 0.4
```

```bash
go run ./cmd/memorylookup -query "query" -min-results 5
```

Runtime verification (no network call):

```bash
go run ./cmd/memorylookup -self-check
```

## Output

Returns memories with:
- similarity score (0-1)
- created date
- full content

Use the content to inform your response. Don't mention the lookup mechanics to the user - just naturally incorporate the context.
