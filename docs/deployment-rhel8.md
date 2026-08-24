# E_AI_Art 单容器部署（RHEL/CentOS 8）

本部署包把 Nginx 前端、Go 后端、MongoDB 7 和 RabbitMQ 3.13 放在同一个镜像中。宿主机只开放 TCP 8081，访问地址为：

```
http://<服务器局域网IP>:8081/
```

## 1. 安装 Docker

在服务器上确认架构：

```
uname -m
# 期望 x86_64
```

安装 Docker CE、Buildx 和 Compose 插件：

```
sudo dnf install -y dnf-plugins-core
sudo dnf config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo dnf install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo systemctl enable --now docker
sudo usermod -aG docker "$USER"
# 重新登录一次，使 docker 组权限生效
docker version
docker compose version
```

如果服务器使用 RHEL 的订阅仓库而不是 CentOS 仓库，请使用贵单位批准的 Docker CE 镜像源；不要同时启动 Podman 的 Docker 兼容 socket。

## 2. 获取代码和配置

```
git clone <你的 GitHub 仓库地址> E_AI_Art
cd E_AI_Art
cp .env.deploy.example .env.deploy
chmod 600 .env.deploy
```

编辑 \`.env.deploy\`，替换所有 \`<replace-with-...>\` 占位符。至少需要填写 AI 接口、MongoDB/RabbitMQ 密码、JWT_SECRET、管理员账号密码。可用下面命令生成 JWT 密钥：

```
openssl rand -hex 32
```

不要把 \`.env.deploy\`、API Key 或数据库密码提交到 Git。

## 3. 首次部署

服务器可以访问 GitHub/Docker Hub 时，直接构建并启动：

```
docker compose --env-file .env.deploy -f compose.deploy.yaml up -d --build
docker compose --env-file .env.deploy -f compose.deploy.yaml ps
docker compose --env-file .env.deploy -f compose.deploy.yaml logs -f e-ai-art
```

看到容器状态为 \`healthy\` 后，在局域网浏览器打开 \`http://服务器IP:8081/\`。后端的 \`/health\`、MongoDB、RabbitMQ 均在容器内部，不应通过宿主机端口访问。

也可以使用仓库脚本：

```
./scripts/build-image.sh
docker compose --env-file .env.deploy -f compose.deploy.yaml up -d
```

## 4. 局域网防火墙

只允许整个内网访问：

```
sudo firewall-cmd --permanent --add-port=8081/tcp
sudo firewall-cmd --reload
sudo firewall-cmd --list-ports
```

若只允许指定网段，先删除上面的通用端口规则，再添加富规则（示例为 192.168.1.0/24）：

```
sudo firewall-cmd --permanent --remove-port=8081/tcp
sudo firewall-cmd --permanent --add-rich-rule='rule family=ipv4 source address=192.168.1.0/24 port port=8081 protocol=tcp accept'
sudo firewall-cmd --reload
```

## 5. 更新和排障

```
git pull
docker compose --env-file .env.deploy -f compose.deploy.yaml up -d --build
docker compose --env-file .env.deploy -f compose.deploy.yaml ps
docker compose --env-file .env.deploy -f compose.deploy.yaml logs --tail=200 e-ai-art
docker compose --env-file .env.deploy -f compose.deploy.yaml restart
```

不要使用 \`docker compose down -v\`，它会删除 MongoDB 和 RabbitMQ 数据卷。普通的 \`down\` 不会删除卷。

首次初始化后，MongoDB 和 RabbitMQ 账号已经写入数据卷；直接修改 \`.env.deploy\` 中的账号密码不会修改已有账号。应先用数据库/队列管理命令完成密码变更，再同步环境文件。

## 6. 离线搬运镜像

在能访问 Docker Hub 的机器上构建并导出：

```
cp .env.deploy.example .env.deploy
# 填入真实配置
./scripts/build-image.sh
./scripts/export-image.sh e-ai-art-local.tar.gz
```

把 \`e-ai-art-local.tar.gz\`、\`compose.deploy.yaml\` 和 \`.env.deploy\` 复制到目标服务器，然后导入并启动：

```
./scripts/import-image.sh e-ai-art-local.tar.gz
docker compose --env-file .env.deploy -f compose.deploy.yaml up -d
```

## 7. 数据备份与恢复

创建 MongoDB 归档：

```
set -a
. ./.env.deploy
set +a
mkdir -p backups
docker compose --env-file .env.deploy -f compose.deploy.yaml exec -T e-ai-art \
  mongodump --host 127.0.0.1 --port 27017 \
  --username "$MONGO_ROOT_USER" --password "$MONGO_ROOT_PASS" \
  --authenticationDatabase admin --archive=/tmp/imagegen.archive
container_id="$(docker compose -f compose.deploy.yaml ps -q e-ai-art)"
docker cp "$container_id:/tmp/imagegen.archive" "backups/imagegen-$(date +%F).archive"
```

恢复前先停止应用写入，再把归档复制进容器并执行：

```bash
docker cp backups/imagegen-YYYY-MM-DD.archive "$container_id:/tmp/imagegen.archive"
docker compose --env-file .env.deploy -f compose.deploy.yaml exec -T e-ai-art \
  mongorestore --host 127.0.0.1 --port 27017 \
  --username "$MONGO_ROOT_USER" --password "$MONGO_ROOT_PASS" \
  --authenticationDatabase admin --archive=/tmp/imagegen.archive --drop
```

恢复前请保留当前数据卷快照，并确认归档版本与镜像版本匹配。
