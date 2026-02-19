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

Run a TypeScript snippet to search memories:

```bash
npx tsx -e "
import { MemoryManager } from './src/memory.js';

const m = new MemoryManager();
const results = await m.search('YOUR_QUERY_HERE');
for (const r of results) {
  console.log(\`[\${r.similarity.toFixed(2)}] \${r.createdAt.slice(0, 10)}\`);
  console.log(r.content);
  console.log('---');
}
"
```

Replace `YOUR_QUERY_HERE` with your search query. Be specific - the semantic search works better with detailed queries.

## Settings

Default: threshold=0.3, always returns at least 3 results (top matches even if below threshold).

```typescript
const results = await m.search('query', { threshold: 0.4 });  // stricter matching
const results = await m.search('query', { minResults: 5 });   // more fallback results
```

## Output

Returns memories with:
- similarity score (0-1)
- created date
- full content

Use the content to inform your response. Don't mention the lookup mechanics to the user - just naturally incorporate the context.
