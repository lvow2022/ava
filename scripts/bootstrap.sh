#!/usr/bin/env bash
set -euo pipefail

echo "installing go tools..."
go mod tidy

echo "done"
