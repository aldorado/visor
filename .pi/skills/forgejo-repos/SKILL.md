---
name: forgejo-repos
description: Use when asked to list repos, check commits/issues, or manage repos on visor's Forgejo instance.
user-invocable: true
---

# Forgejo Repos

Query and manage repos on visor's self-hosted Forgejo.

## Setup

Read token and config from repository-local paths:

```bash
TOKEN=$(cat data/levelups/forgejo/visor-push.token 2>/dev/null)
PORT=$(grep 'FORGEJO_HOST_PORT' .levelup.env 2>/dev/null | cut -d= -f2)
PORT=${PORT:-3002}
ADMIN=$(grep 'FORGEJO_ADMIN_USER' .levelup.env 2>/dev/null | cut -d= -f2)
ADMIN=${ADMIN:-visor}
BASE="http://localhost:${PORT}/api/v1"
```

If token file is missing/empty, Forgejo is not bootstrapped.

## Common queries

### List repos
```bash
curl -s -H "Authorization: token $TOKEN" "$BASE/repos/search?limit=50" | jq -r '.data[] | "\(.full_name) — \(.description // "no description")"'
```

### Recent commits
```bash
REPO="visor"
curl -s -H "Authorization: token $TOKEN" "$BASE/repos/$ADMIN/$REPO/commits?limit=10" | jq -r '.[] | "\(.sha[:8]) \(.commit.message | split("\n")[0])"'
```

### Open issues
```bash
REPO="visor"
curl -s -H "Authorization: token $TOKEN" "$BASE/repos/$ADMIN/$REPO/issues?type=issues&state=open&limit=20" | jq -r '.[] | "#\(.number) \(.title)"'
```

### Create issue
```bash
REPO="visor"
curl -s -X POST -H "Authorization: token $TOKEN" -H "Content-Type: application/json" \
  "$BASE/repos/$ADMIN/$REPO/issues" \
  -d '{"title":"Issue title","body":"Description"}'
```

## Notes

- Forgejo API is local on `http://localhost:$PORT`
- Web UI is via proxy subdomain if enabled
- push-to-create is enabled for git remote flow
