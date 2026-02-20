---
id: TASK-1
title: 'M3: design local parquet ↔ remote sync protocol'
status: To Do
assignee: []
created_date: '2026-02-20 12:37'
labels:
  - m3
  - memory
  - sync
dependencies: []
references:
  - visor.forge.md
priority: high
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Define the sync contract for memories/sessions so visor can keep local parquet as source runtime store while syncing to remote object storage (S3/R2/custom).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 sync model documented for memories and sessions
- [ ] #2 supports append-only row strategy without full file rewrite
- [ ] #3 failure/rollback behavior documented
- [ ] #4 compatible with current local-first runtime
<!-- AC:END -->
