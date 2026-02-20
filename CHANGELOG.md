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

### changed
- repository presentation shifted from execution-board style to public project README

### fixed
- `gofmt` formatting in 6 source files
