---
name: forgejo-repos
description: Use when asked to list repos, check recent commits, view issues, or manage repos on visor's Forgejo instance. Triggered by "zeig mir meine repos", "forgejo repos", "recent commits", "list repos", "forgejo issues", or any question about what's in Forgejo.
user-invocable: true
---

# Forgejo Repos

Query and manage repos on visor's self-hosted Forgejo instance.

## Setup

Read the token and port from levelup config:

```bash
TOKEN=$(cat /root/code/visor/data/levelups/forgejo/visor-push.token 2>/dev/null)
PORT=$(grep 'FORGEJO_HOST_PORT' /root/code/visor/.levelup.env 2>/dev/null | cut -d= -f2)
PORT=${PORT:-3002}
ADMIN=$(grep 'FORGEJO_ADMIN_USER' /root/code/visor/.levelup.env 2>/dev/null | cut -d= -f2)
ADMIN=${ADMIN:-visor}
BASE="http://localhost:${PORT}/api/v1"
```

If token file is empty or missing, Forgejo hasn't been bootstrapped yet — tell the user.

## Common queries

### List repos
```bash
curl -s -H "Authorization: token $TOKEN" "$BASE/repos/search?limit=50" | jq -r '.data[] | "\(.full_name) — \(.description // "no description")"'
```

### Recent commits for a repo
```bash
REPO="visor"  # change as needed
curl -s -H "Authorization: token $TOKEN" "$BASE/repos/$ADMIN/$REPO/commits?limit=10" | jq -r '.[] | "\(.sha[:8]) \(.commit.message | split("\n")[0])"'
```

### Open issues for a repo
```bash
curl -s -H "Authorization: token $TOKEN" "$BASE/repos/$ADMIN/$REPO/issues?type=issues&state=open&limit=20" | jq -r '.[] | "#\(.number) \(.title)"'
```

### Create an issue
```bash
curl -s -X POST -H "Authorization: token $TOKEN" -H "Content-Type: application/json" \
  "$BASE/repos/$ADMIN/$REPO/issues" \
  -d '{"title":"Issue title","body":"Description"}'
```

### Push a repo's README via API (if not yet pushed via git)
```bash
CONTENT=$(base64 -w0 README.md)
curl -s -X POST -H "Authorization: token $TOKEN" -H "Content-Type: application/json" \
  "$BASE/repos/$ADMIN/$REPO/contents/README.md" \
  -d "{\"message\":\"init: add README\",\"content\":\"$CONTENT\"}"
```

## Notes

- Forgejo runs at `http://localhost:$PORT` (only from visor host, not docker containers)
- Web UI available at `https://git.visor.PROXY_DOMAIN` (if proxy level-up is enabled)
- Push-to-create is enabled: `git push forgejo HEAD:main` to a new repo URL creates it automatically
- Token is auto-generated on first start — stored in `data/levelups/forgejo/visor-push.token`
- Configure Forgejo webhook to `https://your-domain.com/forgejo/webhook` for push/PR notifications
