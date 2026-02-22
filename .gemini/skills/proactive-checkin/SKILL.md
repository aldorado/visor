---
name: proactive-checkin
description: Internal scheduled task - sends high-value proactive messages with strict anti-spam rules.
---

# Proactive Check-in Task

You run as a background task.
Send thoughtful proactive messages without being annoying.

## Core behavior

- max 1-2 proactive messages/day
- allowed types: reminder, idea, research-find
- priority: creative ideas > project progress > everything else

## Guardrails

Do NOT send if any is true:
- local time 00:00-08:00 (Europe/Vienna)
- 2 proactive messages already sent today
- less than 4 hours since last proactive message
- active chat in last 90 minutes

If blocked:
- do housekeeping
- log `decision: skipped (guardrail)`
- set `response_text` to empty string

## Candidate quality

Generate 3-5 candidates, score each:
- relevance (0-3)
- novelty (0-2)
- actionability (0-2)

Send only if top score >= 5.
Otherwise skip and log why.

## Context sources

Always check:
1. `data/proactive-log.md`
2. recent sessions in `data/sessions/`
3. semantic memory lookup (memory-lookup skill)
4. Obsidian ideas/forge only if Obsidian level-up is configured:
   - read `OBSIDIAN_VAULT_PATH` from `.levelup.env`
   - use `$OBSIDIAN_VAULT_PATH/ideas/` and `$OBSIDIAN_VAULT_PATH/forge/`
   - if missing, skip Obsidian source cleanly

## Housekeeping

Per run max:
- memory cleanup: 2 actions
- instructions cleanup: 1 tiny improvement in `.pi/SYSTEM.md` only if clearly needed

## Log format

Append short entries to `data/proactive-log.md`.

If sending:
- `response_text`: short natural message
- `conversation_finished`: false

If not sending:
- `response_text`: ""
- `conversation_finished`: true
