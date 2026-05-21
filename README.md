<p align="center">
  <h1 align="center">Smart Recruit</h1>
  <p align="center">基于 gRPC 微服务架构的智能招聘平台，集成 Eino AI 框架驱动的人才对话与数据分析</p>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white" alt="Go version">
  <img src="https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vue.js&logoColor=white" alt="Vue version">
  <img src="https://img.shields.io/badge/gRPC-protocol-2da7b0" alt="gRPC">
  <img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License">
</p>

---

## 功能特性

**HR 管理端**
- 岗位发布与管理，支持部门、地点多维分类
- 候选人台账，投递全流程跟踪（筛选 → 面试 → 录用）
- AI 数据助手：自然语言提问，实时查询招聘数据并生成分析洞察
- 数据看板：招聘漏斗、岗位统计等图表可视化
- 邀请码管理，控制候选人注册权限

**候选人端**
- 岗位浏览与搜索，查看职位详情
- 个人档案管理，简历上传（PDF / DOCX）
- 投递状态实时追踪
- AI 求职助手：基于岗位上下文的智能问答

**平台能力**
- JWT 身份认证与角色鉴权（HR / Candidate）
- 简历文件通过预签名 URL 直传腾讯云 COS，服务端不落盘
- 事务消息 Outbox 模式保证通知投递可靠性
- Docker Compose 一键部署，含前后端及全部中间件

## 系统架构

```
┌─────────────────┐  ┌─────────────────┐
│   hr-frontend    │  │  user-frontend   │
│   (Vue 3 · Vite) │  │  (Vue 3 · Vite)  │
└────────┬────────┘  └────────┬────────┘
         │                    │
         └──────────┬─────────┘
                    │ HTTP REST
         ┌──────────▼─────────┐
         │  web-gin-service   │  ← Gin 网关：JWT 校验、参数绑定、角色鉴权
         │  (Go · Gin)        │
         └──────────┬─────────┘
                    │ gRPC
         ┌──────────▼─────────┐
         │ logic-grpc-service │  ← 核心业务：认证、岗位、投递、AI 对话
         │ (Go · gRPC · Eino) │
         └──┬──────┬──────┬───┘
            │      │      │
     ┌──────▼┐ ┌──▼──┐ ┌▼──────┐
     │ MySQL │ │Redis│ │RabbitMQ│
     └───────┘ └─────┘ └───────┘
```

- **Web 层**（Gin）：只负责 HTTP → gRPC 协议转换，零业务逻辑
- **Logic 层**（gRPC）：承载全部业务逻辑，通过 Eino 框架调用大模型
- **消息队列**：事务 Outbox 模式保障通知可靠投递，简历解析异步化
- **文件存储**：腾讯云 COS 私有 Bucket，预签名 URL 直传

## 技术栈

| 层次 | 技术选型 |
|------|----------|
| HR 前端 | Vue 3 + Vite + Pinia + Element Plus |
| 用户前端 | Vue 3 + Vite + Pinia + Element Plus |
| Web 网关 | Go + Gin + JWT |
| 业务服务 | Go + gRPC + Protobuf |
| AI 框架 | Eino（字节跳动开源） |
| 数据库 | MySQL 8.x |
| 缓存 | Redis 7 |
| 消息队列 | RabbitMQ |
| 对象存储 | 腾讯云 COS |
| 容器化 | Docker + Docker Compose |

## 快速开始

### 前置条件

- **Docker Compose**（推荐）：Docker >= 20.10，docker-compose >= 2.0
- **本地开发**：Go >= 1.21，Node.js >= 18，pnpm，MySQL 8.0，Redis 7，RabbitMQ 3.x

### Docker Compose 部署（推荐）

```bash
# 1. 克隆项目
git clone https://github.com/your-username/smart-recruit.git
cd smart-recruit

# 2. 配置密钥
cp docker/.env.example docker/.env
# 编辑 docker/.env，填写 AI API Key 和 OSS 密钥（必填项见文件注释）

# 3. 一键启动
cd docker
docker-compose up -d --build
```

首次构建约 3-5 分钟，启动后访问：

| 服务 | 地址 |
|------|------|
| HR 管理端 | http://localhost:5173 |
| 候选人用户端 | http://localhost:5174 |
| Web API | http://localhost:8080 |
| RabbitMQ 管理 | http://localhost:15672 |

### 本地开发

```bash
# 1. 初始化数据库
mysql -u root -p < db.sql

# 2. 配置 Logic 服务
cp logic-grpc-service/config/config.example.yaml logic-grpc-service/config/config.yaml
# 编辑 config.yaml，填写 MySQL DSN、Redis、RabbitMQ、OSS 和 AI 配置

# 3. 启动 Logic gRPC 服务（必须先启动）
cd logic-grpc-service
go mod tidy
go run main.go
# 监听 :50051

# 4. 启动 Web Gin 服务
cd web-gin-service
go mod tidy
go run main.go
# 监听 :8080

# 5. 启动前端
cd hr-frontend && pnpm install && pnpm run dev   # → localhost:5173
cd user-frontend && pnpm install && pnpm run dev  # → localhost:5174
```

## 项目结构

```
smart-recruit/
├── hr-frontend/                # HR 管理端前端 (Vue 3 + Element Plus)
│   └── src/
│       ├── api/                # API 请求层
│       ├── components/         # 通用组件
│       ├── views/hr/           # HR 页面（工作台、岗位管理、AI 对话等）
│       ├── stores/             # Pinia 状态管理
│       └── router/             # 路由定义
├── user-frontend/              # 候选人用户端前端 (Vue 3 + Element Plus)
│   └── src/
│       ├── api/                # API 请求层
│       ├── views/candidate/    # 候选人页面（岗位列表、投递、简历等）
│       └── ...
├── web-gin-service/            # Gin Web 网关服务
│   ├── handler/                # HTTP 处理器（candidate / hr 分组）
│   ├── middleware/             # JWT、限流、CSP 等中间件
│   ├── router/                 # 路由注册
│   └── rpc/                    # gRPC 客户端连接
├── logic-grpc-service/         # 核心业务 gRPC 服务
│   ├── ai/                     # Eino AI 客户端与工具定义
│   ├── service/                # 业务逻辑层
│   ├── repository/             # 数据访问层
│   ├── model/                  # 数据模型
│   ├── migrations/             # 数据库迁移脚本
│   ├── mq/                     # RabbitMQ 发布/消费
│   ├── oss/                    # 腾讯云 COS 客户端
│   └── proto/                  # Protobuf 定义
├── docker/                     # Dockerfiles 与 Compose 编排
├── deploy/k8s/                 # Kubernetes 部署清单
├── db.sql                      # 数据库初始化脚本
├── api.md                      # API 接口文档
├── db.md                       # 数据库设计文档
└── answer.md                   # 架构设计问答
```

## 配置说明

所有密钥通过配置文件注入，已加入 `.gitignore`，不会提交到仓库：

| 配置文件 | 用途 |
|----------|------|
| `logic-grpc-service/config/config.yaml` | MySQL / Redis / RabbitMQ / OSS / AI 密钥 |
| `docker/.env` | Docker Compose 环境变量 |

**COS 配置要点**：Bucket 必须为私有读写，关闭公开访问，CORS 配置允许前端直传。

**AI 配置**：当前使用阿里云百炼兼容 OpenAI 接口，可替换为任意 OpenAI 兼容服务。

```yaml
ai:
  api_key: "sk-xxx"
  model: "qwen-plus"
  base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"
  timeout: "90s"
```

## API 文档

详见 [api.md](./api.md)，在线 Swagger 文档可在启动 Web 服务后访问 `/swagger/index.html`。

## 贡献指南

1. Fork 本仓库
2. 创建特性分支：`git checkout -b feature/amazing-feature`
3. 提交代码：`git commit -m 'feat: add amazing feature'`
4. 推送到远端：`git push origin feature/amazing-feature`
5. 提交 Pull Request

提交信息请遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范。

## 许可证

本项目基于 MIT 许可证开源。
