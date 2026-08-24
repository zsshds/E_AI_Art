#!/usr/bin/env bash
set -euo pipefail

archive="${1:-}"
test -n "$archive" || { printf 'Usage: %s IMAGE.tar.gz\n' "$0" >&2; exit 2; }
test -f "$archive" || { printf 'Image archive not found: %s\n' "$archive" >&2; exit 1; }
gzip -dc "$archive" | docker image load

