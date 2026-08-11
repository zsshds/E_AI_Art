#!/usr/bin/env bash
set -euo pipefail

as_rabbitmq() {
  runuser -u rabbitmq -- "$@"
}

start_rabbitmq() {
  as_rabbitmq rabbitmq-server -detached
  for _ in $(seq 1 60); do
    if as_rabbitmq rabbitmq-diagnostics -q ping >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done

  printf 'Timed out waiting for RabbitMQ bootstrap server\n' >&2
  return 1
}

user_exists() {
  as_rabbitmq rabbitmqctl list_users | awk -F '\t' -v user="$RABBITMQ_USER" '$1 == user { found = 1 } END { exit !found }'
}

start_rabbitmq

if ! user_exists; then
  as_rabbitmq rabbitmqctl add_user "$RABBITMQ_USER" "$RABBITMQ_PASS"
  as_rabbitmq rabbitmqctl set_user_tags "$RABBITMQ_USER" administrator
  as_rabbitmq rabbitmqctl set_permissions -p / "$RABBITMQ_USER" '.*' '.*' '.*'
fi

as_rabbitmq rabbitmqctl stop
