# changelog

all notable changes to this project should be documented in this file.

this project follows a keep-a-changelog style with semantic version tags.

## policy

- add entries for user-visible behavior changes, api changes, and operational changes
- group entries under: `added`, `changed`, `fixed`, `removed`
- do not mix unrelated changes into one release section
- every release gets a date and git tag

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

## unreleased

### added
- local pre-push gate (`scripts/check.sh` + `.githooks/pre-push`): gofmt, go vet, go test -race
- pre-release checklist (`docs/release-checklist.md`)
- semantic versioning policy + tagging flow in `CONTRIBUTING.md`
- m8a documentation baseline for publication readiness (`README`, config reference, operations runbook)
- docs sync marker in `visor.forge.md` to explicitly track when implementation changes are mirrored into roadmap-facing docs
- process reminder in forge notes: implementation change -> forge log update + README status note + changelog entry for user-visible impact

### changed
- repository presentation shifted from execution-board style to public project README
- `README.md` now explicitly notes that M12 iteration 3 included post-research hardening (`validate_openai` setup action + `levelup_preset = "recommended"`)
- `visor.forge.md` now includes a 2026-02-21 sync note tying recent setup hardening changes back to the project timeline

### fixed
- `gofmt` formatting in 6 source files
