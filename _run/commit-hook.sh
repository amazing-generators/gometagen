#!/usr/bin/env bash

set -Eeuo pipefail

echo "[HOOK]" "Commit"

run_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root_path=$(cd "$run_dir/.." && pwd)

cd "$root_path"

VERSION=$(go run ./cmd/gometagen version print -source "$run_dir/values.yml")
BRANCH=$(go run ./cmd/gometagen git branch -source "$root_path")

printf "%s [%s]\n\n%s" "$BRANCH" "$VERSION" "$(cat "$1")" > "$1"

#############################################################################
go test -v ./...

#############################################################################
exit 0
