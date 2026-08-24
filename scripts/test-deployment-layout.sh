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
grep -Eq '^[[:space:]]*restart:[[:space:]]*unless-stopped[[:space:]]*(#.*)?$' compose.deploy.yaml || fail 'container must restart unless stopped'
grep -Eq '^[[:space:]]*-[[:space:]]*mongo_data:/data/db[[:space:]]*(#.*)?$' compose.deploy.yaml || fail 'Mongo data must use the mongo_data volume'
grep -Eq '^[[:space:]]*-[[:space:]]*rabbitmq_data:/var/lib/rabbitmq[[:space:]]*(#.*)?$' compose.deploy.yaml || fail 'RabbitMQ data must use the rabbitmq_data volume'
grep -Eq '^[[:space:]]*healthcheck:[[:space:]]*(#.*)?$' compose.deploy.yaml || fail 'container must define a healthcheck'
for program in mongod rabbitmq backend nginx; do
  grep -Fq "[program:$program]" deploy/supervisord.conf || fail "supervisord must manage $program"
done
grep -Fq 'MONGO_URI=mongodb://$MONGO_APP_USER:$MONGO_APP_PASS@127.0.0.1:27017/imagegen' .env.deploy.example || fail 'example must document loopback Mongo URI'
grep -Fq 'RABBITMQ_URI=amqp://$RABBITMQ_USER:$RABBITMQ_PASS@127.0.0.1:5672/' .env.deploy.example || fail 'example must document loopback RabbitMQ URI'
test -x scripts/build-image.sh || fail 'build script must be executable'
test -x scripts/export-image.sh || fail 'export script must be executable'
test -x scripts/import-image.sh || fail 'import script must be executable'
test -f docs/deployment-rhel8.md || fail 'RHEL 8 deployment guide is missing'
grep -Fq 'docker image save' scripts/export-image.sh || fail 'export script must use docker image save'
grep -Fq 'docker image load' scripts/import-image.sh || fail 'import script must use docker image load'
printf 'PASS: deployment layout checks\n'
