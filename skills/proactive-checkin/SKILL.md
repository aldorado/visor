---
name: proactive-checkin
description: Internal scheduled task - sends high-value proactive messages with strict anti-spam rules.
---

# Proactive Check-in Task

You are running as a background task.
Your job: send thoughtful proactive messages *without being annoying*.

## Core behavior

Target cadence:
- send at most 1-2 proactive messages per day

Allowed proactive message types:
- reminder
- idea
- research-find

Priority order:
1. creative ideas
2. project progress
3. everything else

## Hard guardrails (must obey)

Do NOT send if any is true:
- current local time is between 00:00 and 08:00 (Europe/Vienna)
- 2 proactive messages were already sent today
- less than 4 hours passed since the last proactive message you sent
- there was an active chat recently (user or assistant message in last 90 minutes)

If blocked by guardrails:
- do housekeeping and log `decision: skipped (guardrail)`
- set `response_text` to empty string

## Candidate quality

Before sending, generate 3-5 candidate messages from context, then score each:
- relevance to user's current projects/interests (0-3)
- novelty/freshness (0-2)
- actionability (0-2)

Total score range: 0-7.
Only send if top candidate score >= 5.
Otherwise skip and log why.

## Context sources

Always check:
1. `data/proactive-log.md` (last 24h)
2. recent sessions in `data/sessions/`
3. semantic memory lookup (memory-lookup skill)
4. obsidian ideas + forge plans in:
   - `/root/obsidian/sibwax/ideas/`
   - `/root/obsidian/sibwax/forge/`

Favor open threads, unfinished items, and concrete next steps.

## Housekeeping (light)

Do a small maintenance pass each run:
- memory cleanup: max 2 actions (dedupe/outdated/merge)
- instructions cleanup: max 1 tiny improvement in `.pi/SYSTEM.md` only if clearly needed

## Log format

Write to `data/proactive-log.md` and keep entries short.

```
# proactive check-in log

entries below are auto-cleaned to last 24 hours

---

## 2026-02-18 14:30
*checked:* sessions, memories, obsidian ideas/forge
*guardrails:* day_count=1, since_last_proactive=5h12m, active_chat=false, quiet_hours=false
*candidates:*
- idea: "..." (score 6)
- reminder: "..." (score 4)
- research-find: "..." (score 5)
*decision:* sent (idea, score 6)
*housekeeping:* merged 2 duplicate memories
```

If skipping:

```
*decision:* skipped (guardrail: <reason>)
```

or

```
*decision:* skipped (low value: top score 4)
```

## Output format

If sending:
- `response_text`: short natural message (lowercase)
- `conversation_finished`: false

If not sending:
- `response_text`: ""
- `conversation_finished`: true

## Style

- short, warm, human
- no productivity theater
- one clear thought per message
- no spammy "just checking in" with no value
