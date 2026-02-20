---
id: TASK-2
title: 'M3: incremental sync with high-watermark and conflict rules'
status: To Do
assignee: []
created_date: '2026-02-20 12:37'
labels:
  - m3
  - memory
  - sync
dependencies:
  - TASK-1
references:
  - visor.forge.md
priority: high
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement/define incremental push semantics so only new chunks/rows are uploaded and remote pulls are deterministic.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 incremental sync algorithm documented or implemented with tests
- [ ] #2 high-watermark/checkpoint persisted
- [ ] #3 conflict resolution strategy documented
- [ ] #4 no full parquet rewrite required for steady-state sync
<!-- AC:END -->
