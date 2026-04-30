# AI 生图平台项目

> 本文件用于向 AI 编程助手（Claude Code 等）提供项目背景、架构约定和开发规范。
> 所有参与开发的 AI 助手在生成代码前，必须先阅读本文件。

---

## 项目概述

这是一个面向游戏行业美术团队的**开放式 AI 生图平台**，目标是提供一套工程化的生图流水线，确保批量生图时的**画风一致性**和**分辨率一致性**。

核心诉求：
- 美术可通过前端 UI 配置并锁定风格快照（StyleProfile）
- 普通用户提交自然语言描述，系统自动注入风格锚后生图
- 后端异步处理生图任务，WebSocket 实时推送结果

---

## 技术栈

| 层 | 技术 |
|----|------|
| 前端 | Vue 3 + Vite + TypeScript |
| 后端 | Go（主服务） |
| 生图模型 | OpenAI gpt-image-2（`/v1/images/generations` & `/v1/images/edits`） |
| 消息队列 | RabbitMQ（任务异步分发） |
| 实时推送 | WebSocket（gorilla/websocket） |
| 存储 | OSS / CDN（生图结果） |
| 数据库 | MongoDB（StyleProfile、任务记录） |
| 容器化 | Docker + Docker Compose（全部基础设施本地化） |

---

## 目录结构约定

```
/
├── frontend/                  # Vue 前端
│   ├── src/
│   │   ├── views/
│   │   │   ├── StyleEditor.vue        # 美术风格配置页（核心）
│   │   │   ├── GeneratePage.vue       # 用户生图页
│   │   │   └── ReviewQueue.vue        # 人工审核队列
│   │   ├── components/
│   │   │   ├── StyleForm.vue          # 风格表单组件
│   │   │   ├── PromptPreview.vue      # Prompt 实时预览
│   │   │   └── ImagePreviewGrid.vue   # 4宫格预览组件
│   │   ├── stores/
│   │   │   └── styleProfile.ts        # Pinia store
│   │   └── api/
│   │       └── style.ts               # 接口封装
│   ├── Dockerfile
│   └── nginx.conf                     # 生产用 nginx 配置
│
├── backend/                   # Go 后端
│   ├── cmd/server/            # 启动入口
│   ├── internal/
│   │   ├── handler/           # HTTP handler
│   │   ├── service/
│   │   │   ├── prompt/        # PromptBuilder 核心逻辑
│   │   │   ├── image/         # gpt-image-2 API 封装
│   │   │   └── task/          # 任务队列管理
│   │   ├── model/             # 数据模型（StyleProfile、Task 等）
│   │   ├── repo/              # MongoDB 操作层
│   │   └── ws/                # WebSocket Hub
│   ├── config/                # 配置文件（YAML）
│   └── Dockerfile
│
├── docker/                    # 所有基础设施配置集中在此
│   ├── mongo/
│   │   └── init.js            # 初始化集合索引脚本
│   └── rabbitmq/
│       └── definitions.json   # 预声明 exchange/queue 配置
│
├── docker-compose.yml         # 本地开发一键启动
├── docker-compose.prod.yml    # 生产覆盖配置
├── .env.example               # 环境变量模板（提交到 git）
├── .env                       # 实际环境变量（加入 .gitignore）
└── CLAUDE.md
```

---

## 核心数据模型

### StyleProfile（风格配置快照）

```go
// internal/model/style_profile.go

type StyleProfile struct {
    ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    Name        string             `json:"name" bson:"name"`
    CreatedBy   string             `json:"created_by" bson:"created_by"`
    Version     int                `json:"version" bson:"version"`
    IsLocked    bool               `json:"is_locked" bson:"is_locked"` // 锁定后不可修改，生产使用

    // ── Prompt 语义层（美术填写，最终拼接进正向 Prompt）──
    ArtStyle    string   `json:"art_style" bson:"art_style"`       // 画风核心词，e.g. "anime illustration, cel shading"
    ColorTone   string   `json:"color_tone" bson:"color_tone"`     // 色调，e.g. "warm pastel color palette"
    Lighting    string   `json:"lighting" bson:"lighting"`         // 光照，e.g. "soft diffused lighting"
    QualityTags string   `json:"quality_tags" bson:"quality_tags"` // 质量词，e.g. "highly detailed, clean lineart"
    Composition string   `json:"composition" bson:"composition"`   // 构图，e.g. "centered composition, full body"
    ExtraTokens []string `json:"extra_tokens" bson:"extra_tokens"` // 自定义追加词，美术自由填写

    // ── API 参数层（直接控制 gpt-image-2 请求）──
    Size         string `json:"size" bson:"size"`                   // "1024x1024" | "1536x1024" | "1024x1536"
    APIQuality   string `json:"api_quality" bson:"api_quality"`     // "low" | "medium" | "high"
    Background   string `json:"background" bson:"background"`       // "opaque" | "transparent" | "auto"
    OutputFormat string `json:"output_format" bson:"output_format"` // "png" | "webp" | "jpeg"
    Compression  int    `json:"compression" bson:"compression"`     // 0-100，仅 jpeg/webp 有效

    // ── 风格参考图（用于 Edits 端点，是一致性最强的锁定手段）──
    ReferenceImageURL string `json:"reference_image_url" bson:"reference_image_url"` // OSS 地址

    // ── 最终锁定的 Prompt 模板（美术确认后生成，只读）──
    LockedPromptPrefix string    `json:"locked_prompt_prefix" bson:"locked_prompt_prefix"`
    CreatedAt          time.Time `json:"created_at" bson:"created_at"`
    UpdatedAt          time.Time `json:"updated_at" bson:"updated_at"`
}
```

### Task（生图任务）

```go
// internal/model/task.go

type TaskStatus string

const (
    TaskStatusPending    TaskStatus = "pending"
    TaskStatusProcessing TaskStatus = "processing"
    TaskStatusDone       TaskStatus = "done"
    TaskStatusFailed     TaskStatus = "failed"
)

type Task struct {
    ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    StyleProfileID primitive.ObjectID `json:"style_profile_id" bson:"style_profile_id"`
    UserInput      string             `json:"user_input" bson:"user_input"`     // 用户原始输入
    FinalPrompt    string             `json:"final_prompt" bson:"final_prompt"` // 实际发送给模型的完整 Prompt（用于复现）
    Status         TaskStatus         `json:"status" bson:"status"`
    ResultImageURL string             `json:"result_image_url" bson:"result_image_url"`
    ErrorMessage   string             `json:"error_message" bson:"error_message"`
    RetryCount     int                `json:"retry_count" bson:"retry_count"`
    CreatedBy      string             `json:"created_by" bson:"created_by"`
    CreatedAt      time.Time          `json:"created_at" bson:"created_at"`
    UpdatedAt      time.Time          `json:"updated_at" bson:"updated_at"`
}
```

---

## PromptBuilder — 核心拼接逻辑

**这是整个流水线最核心的代码，所有修改需谨慎。**

```go
// internal/service/prompt/builder.go

type PromptBuilder struct {
    Profile *model.StyleProfile
}

// Build 将风格配置 + 用户输入组合为最终 Prompt
// 顺序：风格锚 > 质量词 > 用户内容
// 风格词放最前，gpt-image-2 遵循自然语言权重，靠前描述影响更大
func (b *PromptBuilder) Build(userInput string) string {
    parts := []string{}
    appendIfNotEmpty := func(s string) {
        if strings.TrimSpace(s) != "" {
            parts = append(parts, s)
        }
    }

    appendIfNotEmpty(b.Profile.ArtStyle)
    appendIfNotEmpty(b.Profile.ColorTone)
    appendIfNotEmpty(b.Profile.Lighting)
    appendIfNotEmpty(b.Profile.QualityTags)
    appendIfNotEmpty(b.Profile.Composition)
    parts = append(parts, b.Profile.ExtraTokens...)

    stylePart := strings.Join(parts, ", ")
    cleaned := b.sanitizeUserInput(userInput)

    if stylePart == "" {
        return cleaned
    }
    return fmt.Sprintf("%s. %s", stylePart, cleaned)
}

// sanitizeUserInput 防止用户输入覆盖风格层
// 移除用户输入中的画风/渲染类关键词，保护 StyleProfile 的权威性
func (b *PromptBuilder) sanitizeUserInput(input string) string {
    blocked := []string{
        "realistic", "photorealistic", "3d render", "3d cg",
        "oil painting", "watercolor", "sketch", "pixel art",
        // 根据项目画风按需扩展
    }
    result := input
    for _, word := range blocked {
        result = strings.ReplaceAll(strings.ToLower(result), word, "")
    }
    return strings.TrimSpace(result)
}

// BuildAPIRequest 组装完整 API 请求结构体
func (b *PromptBuilder) BuildAPIRequest(userInput string) ImageRequest {
    return ImageRequest{
        Model:        "gpt-image-2",
        Prompt:       b.Build(userInput),
        Size:         b.Profile.Size,
        Quality:      b.Profile.APIQuality,
        Background:   b.Profile.Background,
        OutputFormat: b.Profile.OutputFormat,
        N:            1,
    }
}
```

**重要约束：**
- 不得将用户输入直接拼接在风格词之前
- 风格词由 StyleProfile 唯一决定，用户无权修改
- 每次生图都必须将 `FinalPrompt` 记录在 Task 中，用于审计和复现

---

## API 接口设计

### 风格配置相关

```
POST   /api/v1/style-profiles              创建风格配置
GET    /api/v1/style-profiles              列表（美术管理页）
GET    /api/v1/style-profiles/:id          获取单条
PUT    /api/v1/style-profiles/:id          更新（仅未锁定时允许）
PUT    /api/v1/style-profiles/:id/lock     锁定（生产发布，不可逆）
POST   /api/v1/style-profiles/:id/preview  测试生成（同步，返回4张图，仅美术使用）
```

### 生图任务相关

```
POST   /api/v1/tasks                       提交生图任务（异步）→ 返回 taskId
GET    /api/v1/tasks/:id                   查询任务状态
GET    /api/v1/tasks                       历史任务列表
WS     /ws/tasks/:id                       订阅任务结果推送
```

---

## 分辨率预设

**不允许用户自由输入分辨率，统一通过 StyleProfile 绑定预设。**

| 预设名 | 用途 | Size |
|--------|------|------|
| `square_1k` | 头像、技能图标 | 1024×1024 |
| `landscape_hd` | 场景横图、Banner | 1536×1024 |
| `portrait_hd` | 角色立绘 | 1024×1536 |

在 Go 后端维护为常量 map，StyleProfile 只存预设名，BuildAPIRequest 时查表获取实际 size 字符串。

---

## 前端风格配置页（StyleEditor.vue）规范

页面分三个区域，**顺序不可颠倒**：

1. **区域 A — 语义配置表单**：下拉/单选卡片/Tag 输入，对应 StyleProfile 各字段
2. **区域 B — Prompt 实时预览**：Watch 所有表单字段，实时拼接展示最终 Prompt 字符串（只读展示，可手动微调 ExtraTokens）
3. **区域 C — 测试生成面板**：输入测试提示词，调用 `/preview` 接口，展示 4 张对比图，满意后点击「保存并锁定」

**表单字段 UI 组件对应关系：**

| 字段 | 组件类型 | 备注 |
|------|---------|------|
| ArtStyle | 卡片单选（带缩略图） | 至少提供4种预设风格供选择 |
| ColorTone | 色盘 + 文字预设 | 支持自定义输入 |
| Lighting | 下拉选择 | 软光/硬光/逆光/无阴影 |
| APIQuality | Slider（low/medium/high） | 显示对应的延迟和费用提示 |
| Size | 比例图标单选 | 可视化展示1:1/16:9/9:16 |
| ExtraTokens | Tag 输入框 | 支持回车添加，点击删除 |
| ReferenceImageURL | 图片上传 | 上传后存 OSS，是风格一致性最强手段 |

---

## 异步任务流程

```
用户提交 POST /tasks
    │
    ├─ Go Handler 写入 MongoDB（status=pending）
    ├─ 推入 RabbitMQ（Exchange: image_tasks, Queue: task_queue）
    └─ 返回 { taskId }

Worker（goroutine pool 消费 RabbitMQ）
    │
    ├─ 从 MongoDB 读取 StyleProfile
    ├─ PromptBuilder.BuildAPIRequest(userInput)
    ├─ 调用 gpt-image-2 API（最长等待 2min，需设 timeout）
    ├─ 结果上传 OSS
    ├─ 更新 MongoDB Task status=done，写入 result_image_url
    ├─ RabbitMQ ACK 消息
    └─ WebSocket Hub 推送给订阅该 taskId 的客户端

失败处理：
    ├─ 业务失败（API报错）→ NACK + requeue，最多重试2次
    ├─ 超过重试上限 → 投递到 Dead Letter Queue（DLQ），status=failed
    └─ WebSocket 推送失败通知
```

**Worker 并发控制：**
- 使用 goroutine pool 控制最大并发数（建议初始值 5，根据 API rate limit 调整）
- gpt-image-2 复杂 Prompt 最长处理约 2 分钟，超时时间设为 150s
- 失败重试最多 2 次，超出后 status=failed，写入 ErrorMessage

---

## 风格一致性保障机制

按效果从强到弱排序：

1. **参考图锁定**（最强）：StyleProfile 配置参考图后，使用 `/images/edits` 端点而非 `/images/generations`，将参考图作为风格基准
2. **Prompt 前缀保护**：风格词固定在 Prompt 最前，`sanitizeUserInput` 阻止覆盖
3. **StyleProfile 版本快照**：Task 记录生图时所用的 ProfileID + Version，风格更新不影响历史任务
4. **自动质检（可选）**：生图后调用 gpt-4o vision，判断是否符合风格描述，不符合进人工审核队列

---

## 开发阶段规划

| 阶段 | 内容 | 交付物 |
|------|------|--------|
| Week 1 | StyleProfile CRUD API + DB 表设计 | Go 接口可用 |
| Week 2 | PromptBuilder + 前端表单 + 实时预览 | Prompt 拼接可用 |
| Week 3 | 同步预览生图（4张对比图）+ 参考图上传 | 美术可配置风格 |
| Week 4 | 异步任务队列 + WebSocket 推送 | 用户生图流程可用 |
| Week 5 | 质检队列 + 审核后台 + 版本管理 | 生产可用 |

---

## 编码规范

### Go

- 包名遵循 Go 惯例，全小写，不使用下划线
- 所有对外 HTTP 错误必须返回统一结构：`{ "code": int, "message": string, "data": any }`
- gpt-image-2 API Key 通过环境变量注入（`OPENAI_API_KEY`），不得硬编码
- 所有 MongoDB 操作通过 `internal/repo` 层，禁止在 handler/service 中直接调用 mongo driver
- MongoDB 文档 ID 统一使用 `primitive.ObjectID`，不使用自定义字符串 ID
- Worker 中的 API 调用必须有 context timeout，不得使用 `context.Background()` 裸调
- RabbitMQ 消费者必须在业务成功后才 ACK，失败时 NACK + requeue，超重试上限投 DLQ

### Vue

- 使用 Composition API + `<script setup>` 语法
- 状态管理使用 Pinia，按功能模块拆分 store
- API 请求统一封装在 `src/api/` 目录，不在组件中直接使用 `fetch`/`axios`
- 风格表单中所有字段变更必须通过 `watch` 驱动 PromptPreview 实时更新，禁止命令式调用

---

## 已知限制（gpt-image-2）

| 限制项 | 说明 | 应对策略 |
|--------|------|---------|
| 跨次一致性 | 多次生成同一内容风格可能漂移 | 上传参考图使用 Edits 端点 |
| 最大延迟 | 复杂 Prompt 最长约 2 分钟 | 全链路异步 + WebSocket 推送 |
| 无 Seed 控制 | 无法像 SD 一样固定随机种子 | 依赖 Prompt 前缀 + 参考图 |
| 无 LoRA/ControlNet | 无权重级别控制 | 所有风格控制依赖 Prompt 工程 |
| 内容审核 | 内置 moderation，部分游戏内容可能被拦截 | 使用 `moderation: "low"` 参数（需评估合规性） |

---

## Docker 基础设施

**所有基础设施通过 Docker Compose 管理，不在宿主机直接安装任何服务。**

### docker-compose.yml（本地开发）

```yaml
version: "3.9"

services:
  # ── 基础设施 ──────────────────────────────────────────
  mongo:
    image: mongo:7.0
    container_name: imagegen_mongo
    restart: unless-stopped
    ports:
      - "27017:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: ${MONGO_ROOT_USER}
      MONGO_INITDB_ROOT_PASSWORD: ${MONGO_ROOT_PASS}
      MONGO_INITDB_DATABASE: imagegen
    volumes:
      - mongo_data:/data/db
      - ./docker/mongo/init.js:/docker-entrypoint-initdb.d/init.js:ro
    networks:
      - imagegen_net

  rabbitmq:
    image: rabbitmq:3.13-management
    container_name: imagegen_rabbitmq
    restart: unless-stopped
    ports:
      - "5672:5672"    # AMQP 协议端口（Go 后端连接）
      - "15672:15672"  # Management UI（浏览器访问：http://localhost:15672）
    environment:
      RABBITMQ_DEFAULT_USER: ${RABBITMQ_USER}
      RABBITMQ_DEFAULT_PASS: ${RABBITMQ_PASS}
    volumes:
      - rabbitmq_data:/var/lib/rabbitmq
      - ./docker/rabbitmq/definitions.json:/etc/rabbitmq/definitions.json:ro
    networks:
      - imagegen_net

  # ── 应用服务 ──────────────────────────────────────────
  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    container_name: imagegen_backend
    restart: unless-stopped
    ports:
      - "8080:8080"
    env_file:
      - .env
    depends_on:
      - mongo
      - rabbitmq
    networks:
      - imagegen_net

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    container_name: imagegen_frontend
    restart: unless-stopped
    ports:
      - "3000:80"
    depends_on:
      - backend
    networks:
      - imagegen_net

volumes:
  mongo_data:
  rabbitmq_data:

networks:
  imagegen_net:
    driver: bridge
```

### RabbitMQ 队列声明（docker/rabbitmq/definitions.json）

```json
{
  "exchanges": [
    {
      "name": "image_tasks",
      "type": "direct",
      "durable": true
    },
    {
      "name": "image_tasks_dlx",
      "type": "direct",
      "durable": true,
      "comment": "Dead Letter Exchange，超重试上限的任务投递到此"
    }
  ],
  "queues": [
    {
      "name": "task_queue",
      "durable": true,
      "arguments": {
        "x-dead-letter-exchange": "image_tasks_dlx",
        "x-dead-letter-routing-key": "dead_tasks",
        "x-message-ttl": 180000
      }
    },
    {
      "name": "dead_task_queue",
      "durable": true,
      "comment": "死信队列，可在 Management UI 中手动排查"
    }
  ],
  "bindings": [
    { "source": "image_tasks", "destination": "task_queue", "routing_key": "task" },
    { "source": "image_tasks_dlx", "destination": "dead_task_queue", "routing_key": "dead_tasks" }
  ]
}
```

### MongoDB 初始化（docker/mongo/init.js）

```javascript
// 创建应用账号（权限隔离，不使用 root）
db.getSiblingDB("imagegen").createUser({
  user: process.env.MONGO_APP_USER || "imagegen_app",
  pwd: process.env.MONGO_APP_PASS || "changeme",
  roles: [{ role: "readWrite", db: "imagegen" }]
});

const db = db.getSiblingDB("imagegen");

// 创建集合 + 索引
db.createCollection("style_profiles");
db.style_profiles.createIndex({ "created_by": 1 });
db.style_profiles.createIndex({ "is_locked": 1 });
db.style_profiles.createIndex({ "created_at": -1 });

db.createCollection("tasks");
db.tasks.createIndex({ "style_profile_id": 1 });
db.tasks.createIndex({ "status": 1 });
db.tasks.createIndex({ "created_by": 1 });
db.tasks.createIndex({ "created_at": -1 });
// 用于 Worker 查询 pending 任务的复合索引
db.tasks.createIndex({ "status": 1, "created_at": 1 });
```

### Dockerfile — Backend（backend/Dockerfile）

```dockerfile
# 多阶段构建，最终镜像不含 Go 工具链
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
COPY config/ ./config/
EXPOSE 8080
CMD ["./server"]
```

### Dockerfile — Frontend（frontend/Dockerfile）

```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:1.25-alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

### 常用命令

```bash
# 首次启动（构建镜像 + 启动所有服务）
docker compose up -d --build

# 查看所有服务状态
docker compose ps

# 查看后端日志（实时）
docker compose logs -f backend

# 仅重启后端（代码变更后）
docker compose up -d --build backend

# 进入 MongoDB Shell
docker compose exec mongo mongosh -u ${MONGO_ROOT_USER} -p ${MONGO_ROOT_PASS}

# 访问 RabbitMQ 管理界面
# 浏览器打开 http://localhost:15672
# 用户名/密码见 .env 文件

# 停止并清除容器（保留数据卷）
docker compose down

# 停止并清除容器 + 数据卷（完全重置）
docker compose down -v
```

---

## 环境变量

`.env.example` 提交到 git，`.env` 加入 `.gitignore`。

```env
# ── OpenAI ────────────────────────────────────────────
OPENAI_API_KEY=sk-...

# ── MongoDB ───────────────────────────────────────────
MONGO_ROOT_USER=root
MONGO_ROOT_PASS=changeme_root
MONGO_APP_USER=imagegen_app
MONGO_APP_PASS=changeme_app
MONGO_URI=mongodb://imagegen_app:changeme_app@mongo:27017/imagegen
# 注意：容器内使用服务名 "mongo" 而非 "localhost"

# ── RabbitMQ ──────────────────────────────────────────
RABBITMQ_USER=admin
RABBITMQ_PASS=changeme_rabbit
RABBITMQ_URI=amqp://admin:changeme_rabbit@rabbitmq:5672/
# 注意：容器内使用服务名 "rabbitmq" 而非 "localhost"

# ── OSS ───────────────────────────────────────────────
OSS_BUCKET=imagegen-assets
OSS_ENDPOINT=https://oss-cn-hangzhou.aliyuncs.com
OSS_ACCESS_KEY=...
OSS_SECRET_KEY=...

# ── 服务参数 ───────────────────────────────────────────
SERVER_PORT=8080
WORKER_CONCURRENCY=5
IMAGE_TASK_TIMEOUT_SECONDS=150
TASK_MAX_RETRY=2
```
