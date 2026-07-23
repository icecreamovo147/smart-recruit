#!/usr/bin/env bash
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

node scripts/check-backend-boundaries.mjs
node scripts/check-mysql-table-ownership.mjs
node .knowledge/scripts/validate-knowledge.mjs --root .
node .knowledge/scripts/check-references.mjs --root .
