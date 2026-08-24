FROM node:20-bookworm-slim AS frontend-build

WORKDIR /frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-bookworm AS backend-build

WORKDIR /src/backend
COPY backend/go.mod ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM rabbitmq:3.13-management

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl gnupg nginx supervisor \
    && curl -fsSL https://pgp.mongodb.com/server-7.0.asc \
      | gpg --dearmor -o /usr/share/keyrings/mongodb-server-7.0.gpg \
    && echo "deb [signed-by=/usr/share/keyrings/mongodb-server-7.0.gpg] https://repo.mongodb.org/apt/debian bookworm/mongodb-org/7.0 main" \
      > /etc/apt/sources.list.d/mongodb-org-7.0.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends mongodb-org-server mongodb-mongosh \
    && rm -rf /var/lib/apt/lists/* /etc/nginx/sites-enabled/default

COPY --from=frontend-build /frontend/dist/ /usr/share/nginx/html/
COPY --from=backend-build /out/server /app/server
COPY --from=backend-build /src/backend/config/ /app/config/
COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf
COPY deploy/supervisord.conf /app/deploy/supervisord.conf
COPY deploy/mongo-init.js /app/deploy/mongo-init.js
COPY deploy/entrypoint.sh /app/deploy/entrypoint.sh

RUN chmod 0755 /app/deploy/entrypoint.sh \
    && mkdir -p /data/db /var/lib/rabbitmq \
    && chown -R mongodb:mongodb /data/db \
    && chown -R rabbitmq:rabbitmq /var/lib/rabbitmq

EXPOSE 80

ENTRYPOINT ["/app/deploy/entrypoint.sh"]
