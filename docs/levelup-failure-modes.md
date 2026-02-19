# level-up failure modes

this doc is the quick playbook when level-up validation/start fails.

## missing env keys

symptom:
- `missing required env keys: ...`

fix:
- add the missing keys to `.levelup.env`
- re-run `go run ./cmd/visor-admin --project-root . levelup validate`

## missing compose file

symptom:
- `required file missing: ...docker-compose.levelup.<name>.yml`

fix:
- create the file or correct `compose_file` in `levelup.toml`
- ensure path is resolvable relative to base compose file directory

## unknown level-up on enable/disable

symptom:
- `unknown level-up: <name>`

fix:
- confirm a valid `levelups/<name>/levelup.toml` exists
- run `levelup list` to see canonical names

## docker compose config failure

symptom:
- `docker compose config failed: ...`

fix:
- inspect compose syntax and env interpolation errors
- ensure all referenced host paths exist and are readable
- validate base + overlays order

## obsidian mount validation failure

symptom:
- `level-up obsidian mounts: ...`

fix:
- set `OBSIDIAN_CONFIG_PATH` and `OBSIDIAN_VAULT_PATH`
- ensure visor process user can create/write in those directories

## obsidian smoke check failure

symptom:
- `obsidian smoke check failed: ...`

fix:
- verify container is up and port mapping is correct
- set `OBSIDIAN_SMOKE_URL` for explicit probe target
- check local firewall or port collisions

## himalaya command failure

symptom:
- `himalaya envelope list: ...` or `himalaya send: ...`

fix:
- confirm himalaya binary is present/reachable in runtime path
- verify account and IMAP/SMTP credentials
- validate tls and server/port combination

## safe fallback behavior

when a level-up fails validation:
- do not apply compose changes
- keep visor host process running
- return explicit error with actionable message
