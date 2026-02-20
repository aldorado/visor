# Contributing to visor

thanks for contributing.

## development setup

1. install go 1.22+
2. clone the repo
3. copy `.levelup.env.example` to `.levelup.env` and fill required values
4. run tests:

```bash
go test ./...
```

5. run visor locally:

```bash
go run .
```

## code style

- keep code simple and explicit
- prefer small focused changes
- run `gofmt` before committing
- run `go vet ./...` before opening a PR

## pull requests

- one focused topic per PR
- include a short "what changed" and "why"
- mention any config, migration, or behavior change
- keep docs updated when behavior changes

## release safety

before merging:

```bash
gofmt ./...
go vet ./...
go test ./...
```

if any check fails, do not merge.

## versioning

visor uses semantic versioning (`vX.Y.Z`):

- `v0.x.y` — pre-stable. breaking changes can happen between minors.
- `v1.0.0` criteria: telegram webhook stable, at least one agent backend working, memory + voice + scheduling functional, CI green.
- *patch* (`z`): bug fixes, no behavior change
- *minor* (`y`): new features, backward-compatible
- *major* (`x`): breaking changes to config, agent protocol, or stored data format

### tagging a release

```bash
# ensure clean tree + all checks green
gofmt -l .
go vet ./...
go test -race ./...

# tag
git tag -a v0.1.0 -m "release v0.1.0"
git push origin v0.1.0
```

see `docs/release-checklist.md` for the full pre-release checklist.
