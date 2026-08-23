#!/usr/bin/env bash
set -euo pipefail
fail() { printf 'FAIL: %s\n' "$1" >&2; exit 1; }

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/.." && pwd)"
cd "$repo_root"

test -f compose.deploy.yaml || fail 'compose.deploy.yaml is missing'
test -f deploy/Dockerfile || fail 'deploy/Dockerfile is missing'
test -f deploy/entrypoint.sh || fail 'deploy/entrypoint.sh is missing'
test -f deploy/supervisord.conf || fail 'deploy/supervisord.conf is missing'
test -f deploy/nginx.conf || fail 'deploy/nginx.conf is missing'
test -f .env.deploy.example || fail '.env.deploy.example is missing'
grep -Eq "^[[:space:]]*-[[:space:]]*(8081:80|\"8081:80\"|'8081:80')[[:space:]]*(#.*)?$" compose.deploy.yaml || fail 'LAN entry must be 8081:80'

in_ports=0
ports_indent=0
while IFS= read -r line || test -n "$line"; do
  content="${line%%#*}"
  test -z "${content//[[:space:]]/}" && continue

  leading="${content%%[![:space:]]*}"
  if [[ "$content" =~ ^[[:space:]]*ports:[[:space:]]*$ ]]; then
    in_ports=1
    ports_indent=${#leading}
    continue
  fi

  test "$in_ports" -eq 1 || continue
  test "${#leading}" -gt "$ports_indent" || { in_ports=0; continue; }

  if [[ "$content" =~ ^[[:space:]]*(-[[:space:]]*)?(target|published):[[:space:]]*['\"]?(27017|5672|15672|8080)(/(tcp|udp))?['\"]?[[:space:]]*$ ]]; then
    fail 'internal ports must not be published'
  fi

  if [[ "$content" =~ ^[[:space:]]*-[[:space:]]*(.*)$ ]]; then
    port_entry="${BASH_REMATCH[1]}"
    port_entry="${port_entry%"${port_entry##*[![:space:]]}"}"
    if [[ "$port_entry" =~ ^\"(.*)\"$ || "$port_entry" =~ ^\'(.*)\'$ ]]; then
      port_entry="${BASH_REMATCH[1]}"
    fi
    if [[ "$port_entry" =~ (^|:)(27017|5672|15672|8080)(:|/|$) ]]; then
      fail 'internal ports must not be published'
    fi
  fi
done < compose.deploy.yaml

grep -Eq '^[[:space:]]*proxy_pass[[:space:]]+http://127\.0\.0\.1:8080;[[:space:]]*(#.*)?$' deploy/nginx.conf || fail 'nginx must use loopback backend'
printf 'PASS: deployment layout checks\n'
