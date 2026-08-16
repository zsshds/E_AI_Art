#!/usr/bin/env bash
set -euo pipefail
fail() { printf 'FAIL: %s\n' "$1" >&2; exit 1; }

test -f compose.deploy.yaml || fail 'compose.deploy.yaml is missing'
test -f deploy/Dockerfile || fail 'deploy/Dockerfile is missing'
test -f deploy/entrypoint.sh || fail 'deploy/entrypoint.sh is missing'
test -f deploy/supervisord.conf || fail 'deploy/supervisord.conf is missing'
test -f deploy/nginx.conf || fail 'deploy/nginx.conf is missing'
test -f .env.deploy.example || fail '.env.deploy.example is missing'
grep -Fq '8081:80' compose.deploy.yaml || fail 'LAN entry must be 8081:80'
! grep -Eq '(^|[^0-9])(27017|5672|15672|8080):' compose.deploy.yaml || fail 'internal ports must not be published'
grep -Fq 'proxy_pass http://127.0.0.1:8080' deploy/nginx.conf || fail 'nginx must use loopback backend'
printf 'PASS: deployment layout checks\n'
