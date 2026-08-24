#!/usr/bin/env bash
set -euo pipefail

required_vars=(
  MONGO_ROOT_USER
  MONGO_ROOT_PASS
  MONGO_APP_USER
  MONGO_APP_PASS
  RABBITMQ_USER
  RABBITMQ_PASS
  JWT_SECRET
  OPENAI_API_KEY
)

for var_name in "${required_vars[@]}"; do
  if [[ -z "${!var_name:-}" ]]; then
    printf 'Missing required environment variable: %s\n' "$var_name" >&2
    exit 1
  fi
done

export MONGO_URI="mongodb://${MONGO_APP_USER}:${MONGO_APP_PASS}@127.0.0.1:27017/imagegen"
export RABBITMQ_URI="amqp://${RABBITMQ_USER}:${RABBITMQ_PASS}@127.0.0.1:5672/"
export SERVER_PORT=8080
export RABBITMQ_DEFAULT_USER="$RABBITMQ_USER"
export RABBITMQ_DEFAULT_PASS="$RABBITMQ_PASS"

mkdir -p /data/db /var/lib/rabbitmq
chown -R mongodb:mongodb /data/db
chown -R rabbitmq:rabbitmq /var/lib/rabbitmq

is_new_mongo_volume() {
  ! find /data/db -mindepth 1 -maxdepth 1 -not -name lost+found -print -quit | grep -q .
}

wait_for_mongo() {
  for _ in $(seq 1 60); do
    if mongosh --quiet --host 127.0.0.1 --eval 'db.runCommand({ping: 1}).ok' >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done

  printf 'Timed out waiting for temporary MongoDB initialization server\n' >&2
  return 1
}

stop_temporary_mongo() {
  mongosh --quiet --host 127.0.0.1 \
    --username "$MONGO_ROOT_USER" \
    --password "$MONGO_ROOT_PASS" \
    --authenticationDatabase admin \
    --eval 'db.getSiblingDB("admin").shutdownServer()' >/dev/null 2>&1 || true
}

if is_new_mongo_volume; then
  printf 'Initializing MongoDB data volume\n'
  runuser -u mongodb -- mongod --auth --bind_ip 127.0.0.1 --dbpath /data/db --fork --logpath /tmp/mongod-init.log
  trap stop_temporary_mongo EXIT
  wait_for_mongo

  mongosh --quiet --host 127.0.0.1 --eval '
    db.getSiblingDB("admin").createUser({
      user: process.env.MONGO_ROOT_USER,
      pwd: process.env.MONGO_ROOT_PASS,
      roles: [{ role: "root", db: "admin" }]
    });
  '
  mongosh --quiet --host 127.0.0.1 \
    --username "$MONGO_ROOT_USER" \
    --password "$MONGO_ROOT_PASS" \
    --authenticationDatabase admin \
    --file /app/deploy/mongo-init.js
  stop_temporary_mongo
  trap - EXIT
else
  printf 'MongoDB data volume is populated; skipping initialization\n'
fi

exec supervisord -c /app/deploy/supervisord.conf
