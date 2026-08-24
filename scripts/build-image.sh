#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"
env_file="${1:-.env.deploy}"
test -f "$env_file" || { printf 'Missing deployment env file: %s\n' "$env_file" >&2; exit 1; }
docker compose --env-file "$env_file" -f compose.deploy.yaml build e-ai-art

