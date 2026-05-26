#!/usr/bin/env bash

set -Eeuo pipefail

run_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root_dir="$(cd "$run_dir/.." && pwd)"

cd "$root_dir"

go run ./cmd/gometagen git add-commit-hook -source .
go run ./cmd/gometagen git add-push-hook -source .

if [ -f go.work ]; then
  go work sync
fi

# go generate .  # Нет codegen в этом репозитории.
go mod tidy
