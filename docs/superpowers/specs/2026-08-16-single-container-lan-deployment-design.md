# 单容器局域网部署设计

## 目标

为 RHEL/CentOS 8 x86_64 局域网服务器提供一个可重复构建、可离线搬运的单一 Docker 镜像。镜像在同一容器内运行前端 Nginx、Go 后端、MongoDB 和 RabbitMQ；宿主机仅通过 `8081:80` 暴露应用。

## 范围与约束

- 访问入口为 `http://<服务器 IP>:8081/`，不配置公网域名或 TLS。
- MongoDB、RabbitMQ 和后端不发布到宿主机端口，只接受同容器内连接。
- 数据库和消息队列数据必须使用命名卷持久化；删除或替换容器不应删除数据。
- 运行时凭据放在部署环境文件中，镜像和仓库不包含真实 API Key、密码或 JWT 密钥。
- 服务器可访问 GitHub；同时提供镜像导出和导入流程供离线更新使用。

## 方案选择

采用单体运行时镜像而不是原有四容器 Compose 结构。运行时以受支持的 Linux 基础镜像为基础，安装 Nginx、MongoDB、RabbitMQ 和进程管理器；前端静态文件和 Go 二进制由多阶段构建复制进运行时镜像。

使用进程管理器保持四个长期进程运行，并以前置初始化脚本处理首次 MongoDB 用户创建。进程管理器按依赖顺序启动服务；后端在 MongoDB 和 RabbitMQ 可用后启动。任一关键进程异常退出会使容器退出，交由 Docker 的 `unless-stopped` 策略重启。

## 组件与数据流

1. 宿主机的 TCP 8081 映射到容器 Nginx 的 TCP 80。
2. Nginx 提供 Vue 构建产物，并将 `/api/` 和 `/ws/` 反向代理到 `127.0.0.1:8080`。
3. Go 后端连接 `127.0.0.1:27017` 的 MongoDB 和 `127.0.0.1:5672` 的 RabbitMQ。
4. Docker 命名卷挂载到 MongoDB 数据目录和 RabbitMQ 数据目录，应用配置从 `--env-file .env.deploy` 注入。

## 交付物

- `deploy/Dockerfile`：构建单镜像，包含前端、后端和四个运行时服务。
- `deploy/entrypoint.sh` 与 `deploy/supervisord.conf`：首次初始化、启动顺序和进程生命周期管理。
- `compose.deploy.yaml`：一个服务、`8081:80` 映射、两份命名卷、健康检查和重启策略。
- `.env.deploy.example`：不含真实值的部署配置模板。
- `scripts/build-image.sh`、`scripts/export-image.sh`、`scripts/import-image.sh`：构建、离线导出和离线导入操作。
- `docs/deployment-rhel8.md`：Docker 安装、首次启动、升级、备份恢复和故障排查说明。

## 配置与安全

部署环境文件会覆盖后端的 `MONGO_URI`、`RABBITMQ_URI`、`SERVER_PORT`、AI 接口、OSS、JWT 和管理员凭据。MongoDB 与 RabbitMQ 仅绑定 loopback 地址，且 Compose 不发布对应端口。首次启动创建的数据库账号和 RabbitMQ 默认账号会随其数据卷保存；之后修改凭据需通过各自服务的账号管理流程完成，文档会明确这一点。

## 健康检查、失败行为与验证

容器健康检查验证前端经 Nginx 返回成功、后端 `/health` 正常，以及 MongoDB 和 RabbitMQ 监听本地端口。Nginx、后端、MongoDB 或 RabbitMQ 失效时，进程管理器停止容器，Docker 自动重启。

自动验证包含：Compose 配置渲染、镜像构建、容器健康状态、`http://127.0.0.1:8081/` 响应、后端健康接口、宿主机未发布 27017/5672/8080，以及镜像 `save/load` 后能被 Compose 识别。应用现有 Go 单元测试继续作为构建前验证。

## 非目标

- 不提供公网 HTTPS、域名、负载均衡或多节点高可用。
- 不改变应用业务功能或 AI/OSS 的外部依赖。
- 不将数据库备份上传到外部存储；只提供本地备份与恢复命令。
