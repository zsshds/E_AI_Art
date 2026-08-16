# 单容器局域网部署 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 交付一个包含前端、后端、MongoDB 与 RabbitMQ 的单一 Docker 镜像，并通过宿主机 8081 端口安全提供局域网访问。

**Architecture:** `deploy/Dockerfile` 以 Node 和 Go 多阶段构建前后端，再基于 RabbitMQ 运行时镜像安装 MongoDB、Nginx 与 Supervisor。入口脚本在首次启动时初始化 MongoDB，Supervisor 以依赖顺序管理四个长期进程；`compose.deploy.yaml` 仅发布 `8081:80` 并挂载两份持久化卷。

**Tech Stack:** Docker BuildKit/Compose、Node 20、Go 1.25、Nginx、MongoDB 7、RabbitMQ 3.13、Supervisor、Bash。

---

## 文件结构

- `deploy/Dockerfile`：多阶段镜像构建与运行时依赖。
- `deploy/entrypoint.sh`：必填配置验证、MongoDB 首次初始化、启动 Supervisor。
- `deploy/supervisord.conf`：MongoDB、RabbitMQ、后端、Nginx 的进程管理。
- `deploy/nginx.conf`：静态前端与 loopback API/WebSocket 反向代理。
- `deploy/mongo-init.js`：应用账户和索引。
- `compose.deploy.yaml`：唯一服务、8081 映射、命名卷、健康检查。
- `.env.deploy.example`：不含真实凭据的模板。
- `scripts/test-deployment-layout.sh`：不依赖 Docker 守护进程的部署契约测试。
- `scripts/build-image.sh`、`scripts/export-image.sh`、`scripts/import-image.sh`：镜像构建和离线搬运。
- `docs/deployment-rhel8.md`：RHEL 8 部署、升级、备份与恢复说明。

### Task 1: 编写单容器部署契约测试

**Files:**
- Create: `scripts/test-deployment-layout.sh`
- Test: `scripts/test-deployment-layout.sh`

- [ ] **Step 1: 写出会失败的测试**

```bash
#!/usr/bin/env bash
set -euo pipefail
fail() { printf 'FAIL: %s\n' "$1" >&2; exit 1; }

test -f compose.deploy.yaml || fail 'compose.deploy.yaml is missing'
test -f deploy/Dockerfile || fail 'deploy/Dockerfile is missing'
test -f deploy/entrypoint.sh || fail 'deploy/entrypoint.sh is missing'
test -f deploy/supervisord.conf || fail 'deploy/supervisord.conf is missing'
test -f deploy/nginx.conf || fail 'deploy/nginx.conf is missing'
grep -Fq '8081:80' compose.deploy.yaml || fail 'LAN entry must be 8081:80'
! grep -Eq '(^|[^0-9])(27017|5672|15672|8080):' compose.deploy.yaml || fail 'internal ports must not be published'
grep -Fq 'proxy_pass http://127.0.0.1:8080' deploy/nginx.conf || fail 'nginx must use loopback backend'
printf 'PASS: deployment layout checks\n'
```

- [ ] **Step 2: 验证 RED**

Run: `bash scripts/test-deployment-layout.sh`

Expected: `FAIL: compose.deploy.yaml is missing`。

- [ ] **Step 3: 建立最小文件骨架**

```bash
mkdir -p deploy scripts
touch compose.deploy.yaml deploy/Dockerfile deploy/entrypoint.sh deploy/supervisord.conf deploy/nginx.conf .env.deploy.example
chmod +x scripts/test-deployment-layout.sh
```

- [ ] **Step 4: 再次运行测试**

Run: `bash scripts/test-deployment-layout.sh`

Expected: `FAIL: LAN entry must be 8081:80`。

- [ ] **Step 5: 提交测试**

```bash
git add scripts/test-deployment-layout.sh
git commit -m "test: define single-container deployment contract"
```

### Task 2: 实现单镜像运行时和 Compose 配置

**Files:**
- Create: `deploy/Dockerfile`
- Create: `deploy/entrypoint.sh`
- Create: `deploy/supervisord.conf`
- Create: `deploy/nginx.conf`
- Create: `deploy/mongo-init.js`
- Create: `compose.deploy.yaml`
- Create: `.env.deploy.example`
- Modify: `scripts/test-deployment-layout.sh`
- Test: `scripts/test-deployment-layout.sh`

- [ ] **Step 1: 先增加会失败的运行时检查**

在 `PASS` 行之前加入：

```bash
grep -Fq 'restart: unless-stopped' compose.deploy.yaml || fail 'restart policy is required'
grep -Fq 'mongo_data:/data/db' compose.deploy.yaml || fail 'Mongo volume is required'
grep -Fq 'rabbitmq_data:/var/lib/rabbitmq' compose.deploy.yaml || fail 'RabbitMQ volume is required'
grep -Fq 'healthcheck:' compose.deploy.yaml || fail 'healthcheck is required'
grep -Fq '[program:mongod]' deploy/supervisord.conf || fail 'MongoDB process is required'
grep -Fq '[program:rabbitmq]' deploy/supervisord.conf || fail 'RabbitMQ process is required'
grep -Fq '[program:backend]' deploy/supervisord.conf || fail 'backend process is required'
grep -Fq '[program:nginx]' deploy/supervisord.conf || fail 'nginx process is required'
grep -Fq 'MONGO_URI=mongodb://$MONGO_APP_USER:$MONGO_APP_PASS@127.0.0.1:27017/imagegen' .env.deploy.example || fail 'Mongo URI must be local'
grep -Fq 'RABBITMQ_URI=amqp://$RABBITMQ_USER:$RABBITMQ_PASS@127.0.0.1:5672/' .env.deploy.example || fail 'RabbitMQ URI must be local'
```

- [ ] **Step 2: 验证 RED**

Run: `bash scripts/test-deployment-layout.sh`

Expected: 先因 `LAN entry must be 8081:80` 失败；填入端口映射后因 `restart policy is required` 失败。

- [ ] **Step 3: 实现最小配置**

创建 `compose.deploy.yaml`，必须只有 `e-ai-art` 服务、两份命名卷及如下入口：

```yaml
services:
  e-ai-art:
    image: ${IMAGE_NAME:-e-ai-art}:${IMAGE_TAG:-local}
    build:
      context: .
      dockerfile: deploy/Dockerfile
    env_file: .env.deploy
    ports:
      - "8081:80"
    volumes:
      - mongo_data:/data/db
      - rabbitmq_data:/var/lib/rabbitmq
    restart: unless-stopped
volumes:
  mongo_data:
  rabbitmq_data:
```

实现细则：

- `deploy/nginx.conf` 保留 SPA `try_files `uri `uri/ /index.html`，并将 `/api/` 和 `/ws/` 均代理到 `127.0.0.1:8080`。
- `deploy/supervisord.conf` 必须定义 `mongod=10`、`rabbitmq=20`、`backend=30`、`nginx=40` 的优先级，四个程序均设 `autorestart=true`、`startretries=10`，并使用 `stopasgroup=true` 和 `killasgroup=true`。
- `deploy/entrypoint.sh` 必须拒绝空的 `MONGO_ROOT_USER`、`MONGO_ROOT_PASS`、`MONGO_APP_USER`、`MONGO_APP_PASS`、`RABBITMQ_USER`、`RABBITMQ_PASS`、`JWT_SECRET` 和 `OPENAI_API_KEY`；首次数据卷启动临时 MongoDB，执行 `deploy/mongo-init.js`，随后 `exec /usr/bin/supervisord -n -c /etc/supervisor/supervisord.conf`。
- `deploy/Dockerfile` 必须先运行 `npm ci && npm run build` 和 `go build -o server ./cmd/server`，运行时安装 Nginx、Supervisor、MongoDB 7 server/shell 和 `curl`，复制 `frontend/dist`、二进制、`backend/config` 和 `deploy` 文件，使用 `ENTRYPOINT` 启动脚本并只 `EXPOSE 80`。
- `.env.deploy.example` 仅有示例/占位值，且包含镜像名和标签、Mongo/RabbitMQ 凭据、AI/OSS/JWT/管理员配置和任务超时参数。

- [ ] **Step 4: 验证 GREEN**

Run: `bash scripts/test-deployment-layout.sh`

Expected: `PASS: deployment layout checks`。

- [ ] **Step 5: 验证 Compose**

Run: `cp .env.deploy.example .env.deploy && docker compose -f compose.deploy.yaml config && rm .env.deploy`

Expected: 合法 Compose 配置，且没有 27017、5672、15672 或 8080 的 published port。

- [ ] **Step 6: 提交运行时**

```bash
git add deploy compose.deploy.yaml .env.deploy.example scripts/test-deployment-layout.sh
git commit -m "feat: add single-container deployment image"
```

### Task 3: 交付构建、离线搬运和 RHEL 8 文档

**Files:**
- Create: `scripts/build-image.sh`
- Create: `scripts/export-image.sh`
- Create: `scripts/import-image.sh`
- Create: `docs/deployment-rhel8.md`
- Modify: `scripts/test-deployment-layout.sh`
- Test: `scripts/test-deployment-layout.sh`

- [ ] **Step 1: 写出会失败的交付物检查**

在 `PASS` 行之前加入：

```bash
test -x scripts/build-image.sh || fail 'build script must be executable'
test -x scripts/export-image.sh || fail 'export script must be executable'
test -x scripts/import-image.sh || fail 'import script must be executable'
test -f docs/deployment-rhel8.md || fail 'RHEL 8 guide is missing'
grep -Fq 'docker image save' scripts/export-image.sh || fail 'export must use docker image save'
grep -Fq 'docker image load' scripts/import-image.sh || fail 'import must use docker image load'
```

- [ ] **Step 2: 验证 RED**

Run: `bash scripts/test-deployment-layout.sh`

Expected: `FAIL: build script must be executable`。

- [ ] **Step 3: 实现脚本与运维手册**

```bash
# scripts/build-image.sh
docker compose -f compose.deploy.yaml build e-ai-art

# scripts/export-image.sh
docker image save "${IMAGE_NAME:-e-ai-art}:${IMAGE_TAG:-local}" | gzip > "${1:-e-ai-art-image.tar.gz}"

# scripts/import-image.sh
gzip -dc "$1" | docker image load
```

三个脚本都须以 `#!/usr/bin/env bash` 和 `set -euo pipefail` 开始且可执行。`docs/deployment-rhel8.md` 必须明确说明：RHEL 8 Docker/Compose 安装前置、复制环境文件、`docker compose -f compose.deploy.yaml up -d --build` 首次部署、`http://<server-ip>:8081/` 验收、日志、升级、离线 save/load、`mongodump`/`mongorestore` 备份恢复、仅放行 8081 的 firewalld 命令，以及已有持久化数据后不能直接改 Mongo/RabbitMQ 账号密码。

- [ ] **Step 4: 验证 GREEN**

Run: `bash scripts/test-deployment-layout.sh`

Expected: `PASS: deployment layout checks`。

- [ ] **Step 5: 完整验证**

Run: `cd backend && go test ./... && cd .. && docker compose -f compose.deploy.yaml build && cp .env.deploy.example .env.deploy && docker compose -f compose.deploy.yaml up -d && curl --fail http://127.0.0.1:8081/ && docker compose -f compose.deploy.yaml ps && docker compose -f compose.deploy.yaml down && rm .env.deploy`

Expected: Go 测试通过；镜像构建成功；容器 healthy；入口成功响应；仅发布 `8081->80`。

- [ ] **Step 6: 提交交付工具与文档**

```bash
git add scripts docs/deployment-rhel8.md
git commit -m "docs: add RHEL 8 deployment guide"
```

