#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

mkdir -p bin
go build -o bin/visor .
systemctl restart visor
systemctl status visor --no-pager | head -20
