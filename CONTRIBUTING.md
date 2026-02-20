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
