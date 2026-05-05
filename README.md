<p align="center">
  <img src="frontend/public/logo.png" alt="E_AI_Art Logo" width="128" />
</p>

<h1 align="center">E_AI_Art</h1>
<p align="center">面向游戏开发的 AI 美术资源生成平台</p>

---

## 简介

E_AI_Art 是一个面向游戏开发团队的 AI 生图工作站，支持将美术风格沉淀为可复用的"风格配置"，团队成员通过项目共享风格和任务队列，实现批量、一致的游戏美术资产生产。

**核心场景**: 角色设定卡、游戏 UI 面板、技能/道具图标集、关卡场景概念、卡牌插画、像素艺术等。

## 技术栈

| 层级 | 技术 |
|------|------|
| **前端** | Vue 3 + TypeScript + Vite + Pinia + Vue Router |
| **后端** | Go + Echo (Web框架) + MongoDB + RabbitMQ |
| **AI** | GPT-4o Image / GPT-Image-2 / Gemini / Midjourney 多模型支持 |
| **部署** | Docker Compose 一键编排 |

## 功能

- **风格配置**: 管理员创建可复用的美术风格（画风、色调、模型），锁定后供团队使用
- **AI 生图**: 选择风格 + 输入描述 → 一键生成，支持多种分辨率（最高 4K）
- **任务队列**: 团队共享任务队列，实时查看生图进度和结果
- **项目管理**: 管理员创建项目，添加成员，同项目共享风格和任务
- **提示词助手**: 内置游戏场景专属提示词库，一键插入质量增强、风格修饰、负面提示
- **多模型支持**: GPT-4o Image / GPT-Image-2 / Gemini / Nano Banana / Midjourney
- **WebSocket 实时推送**: 生图进度实时更新

## 快速开始

```bash
# 1. 克隆项目
git clone <repo-url> && cd E_AI_Art

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env，填入你的 OPENAI_API_KEY

# 3. 启动所有服务
docker-compose up -d

# 4. 访问
# 前端: http://localhost:3000
# 默认管理员: admin / admin123
```

## 项目结构

```
E_AI_Art/
├── frontend/           # Vue 3 前端
│   ├── src/
│   │   ├── views/      # 页面: 生图/风格配置/任务队列/项目管理/系统设置
│   │   ├── components/ # 组件: StyleForm/ArtStyleSelector/PromptAssistant...
│   │   ├── api/        # API 客户端
│   │   ├── stores/     # Pinia 状态管理
│   │   └── composables/# 组合式函数
│   └── public/         # 静态资源
├── backend/            # Go 后端
│   ├── cmd/server/     # 入口
│   ├── internal/
│   │   ├── handler/    # HTTP 处理器
│   │   ├── model/      # 数据模型
│   │   ├── repo/       # MongoDB 数据访问
│   │   ├── service/    # 业务逻辑 (Prompt构建/任务管理/图片客户端)
│   │   ├── middleware/ # JWT 认证中间件
│   │   └── ws/         # WebSocket Hub
│   └── config/         # 配置文件
└── docker-compose.yml  # 容器编排
```

## 安全

- **绝不**在代码中硬编码 API Key，使用环境变量或 `.env` 文件
- `.env` 和 `.vscode/` 已加入 `.gitignore`
- 默认管理员密码仅用于开发环境，生产环境务必修改
- 生产部署前请更换 `JWT_SECRET`

## License

MIT
