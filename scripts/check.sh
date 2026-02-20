#!/usr/bin/env bash
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

echo "==> gofmt"
unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
  echo "FAIL: needs gofmt:"
  echo "$unformatted"
  exit 1
fi
echo "    ok"

echo "==> go vet"
go vet ./...
echo "    ok"

echo "==> go test"
go test -race -count=1 ./...
echo "    ok"

echo ""
echo "all checks passed"
