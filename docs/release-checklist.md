# pre-release checklist

run through this before tagging any release.

## code quality

- [ ] `gofmt -l .` returns nothing
- [ ] `go vet ./...` passes
- [ ] `go test -race ./...` passes (all packages green)
- [ ] no `TODO` or `FIXME` items blocking this release

## security

- [ ] no secrets in tracked files (`grep -r 'PRIVATE\|SECRET\|PASSWORD\|TOKEN' --include='*.go' --include='*.toml' --include='*.yml'` — only references, no values)
- [ ] `.env` and `.levelup.env` are in `.gitignore`
- [ ] `data/` directory is in `.gitignore`
- [ ] no hardcoded API keys or tokens in source

## docs

- [ ] `README.md` reflects current feature set
- [ ] `docs/config-reference.md` covers all env vars
- [ ] `CHANGELOG.md` has entry for this release
- [ ] `CONTRIBUTING.md` is current

## build

- [ ] `go build .` succeeds with no warnings
- [ ] binary starts and responds on `/health`

## release

- [ ] `scripts/check.sh` passes (or pre-push hook ran clean)
- [ ] git tree is clean (`git status` shows nothing)
- [ ] version tag follows semver (`vX.Y.Z`)
- [ ] tag is annotated: `git tag -a vX.Y.Z -m "release vX.Y.Z"`
