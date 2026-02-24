# changelog

all notable project changes are documented here.

## unreleased

### added
- local pre-push quality gate (`scripts/check.sh` + `.githooks/pre-push`) running `gofmt`, `go vet`, and `go test -race`.
- pre-release checklist in `docs/release-checklist.md`.
- semver/tagging policy documented in `CONTRIBUTING.md`.
- m8a documentation baseline for publication readiness (`README`, config reference, operations runbook).
- docs-sync marker in `visor.forge.md` to track roadmap/docs alignment.
- process reminder: implementation change -> forge log update + README status note + changelog entry (if user-visible).
- `/agent` webhook command to switch runtime backend without redeploying.
- backend selection wiring in agent queue/registry/server flow for per-request routing.
- `.gemini` prompt/skills mirror and prompt-sync support for Gemini metadata.

### changed
- repository presentation moved from execution-board style to public project README style.
- `README.md` now explicitly notes M12 iteration 3 post-research hardening (`validate_openai`, recommended setup preset).
- `visor.forge.md` now includes a 2026-02-21 sync note connecting setup hardening changes to timeline updates.
- webhook skill trigger exposure now respects enabled levelups before surfacing tools/prompts.
- self-restart docs now clarify agent code restarts vs external service manager (`systemd`) restarts.
- auto-restart docs now use generic project paths instead of hardcoded local paths.
- setup skill synchronization restored across canonical `skills/` and runtime prompt directories.
- Gemini agent now logs request start/first-token/done timing to expose real latency in production.
- Gemini backend now reuses the latest session for a 20 minute window (configurable via `GEMINI_RESUME_WINDOW_MINUTES`).

### fixed
- `gofmt` formatting cleanup in 6 source files.
- prompt-sync duplication issue for Gemini caused by temporary `.agents` mirror strategy.
- Gemini stream-json parsing now reads assistant top-level `content` events correctly.
- levelup-creator skill frontmatter parse failure fixed by quoting the `description` field.

### removed
- deprecated `.agents/` mirror tree to prevent duplicated skill metadata and drift.

## format

```md
## [x.y.z] - YYYY-MM-DD
### added
- ...

### changed
- ...

### fixed
- ...

### removed
- ...
```
