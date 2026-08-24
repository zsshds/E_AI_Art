#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"
env_file="${2:-.env.deploy}"
output_file="${1:-e-ai-art-image.tar.gz}"
test -f "$env_file" || { printf 'Missing deployment env file: %s\n' "$env_file" >&2; exit 1; }
image_ref="$(docker compose --env-file "$env_file" -f compose.deploy.yaml config --images | head -n 1)"
test -n "$image_ref" || { printf 'Unable to resolve image reference\n' >&2; exit 1; }
printf 'Exporting %s to %s\n' "$image_ref" "$output_file"
docker image save "$image_ref" | gzip > "$output_file"
