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

### changed
- repository presentation moved from execution-board style to public project README style.
- `README.md` now explicitly notes M12 iteration 3 post-research hardening (`validate_openai`, `levelup_preset = "recommended"`).
- `visor.forge.md` now includes a 2026-02-21 sync note connecting setup hardening changes to timeline updates.

### fixed
- `gofmt` formatting cleanup in 6 source files.

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
