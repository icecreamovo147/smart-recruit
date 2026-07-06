# 04-TEST_COMMANDS — 测试命令集

> 版本：v1.0
> 创建日期：2026-06-26

---

## 1. 命令概览

每个任务完成后必须运行对应模块的测试命令，全部通过方可标记 Done。

---

## 2. 前端命令

### 2.1 hr-frontend

```bash
# 安装依赖
cd hr-frontend && pnpm install

# TypeScript 类型检查
cd hr-frontend && pnpm typecheck     # 实际执行: vue-tsc --noEmit

# 单元测试
cd hr-frontend && pnpm test          # 实际执行: vitest run

# 构建检查
cd hr-frontend && pnpm build         # 实际执行: vite build

# 开发服务器（仅用于手动验证，非自动化命令）
cd hr-frontend && pnpm dev           # → http://localhost:5173
```

### 2.2 user-frontend

```bash
cd user-frontend && pnpm install
cd user-frontend && pnpm typecheck   # vue-tsc --noEmit
cd user-frontend && pnpm test        # vitest run
cd user-frontend && pnpm build       # vite build
# dev: pnpm dev                      # → http://localhost:5174
```

### 2.3 interviewer-frontend

```bash
cd interviewer-frontend && pnpm install
cd interviewer-frontend && pnpm typecheck  # vue-tsc --noEmit
cd interviewer-frontend && pnpm test       # vitest run
cd interviewer-frontend && pnpm build      # vite build
# dev: pnpm dev                            # → http://localhost:5175
```

### 2.4 前端 Lint

> **待确认**：项目 `package.json` 中未定义 `lint` 命令（无 eslint 配置）。如需 lint 检查，需先配置。

---

## 3. 后端命令

### 3.1 logic-grpc-service

```bash
# 安装依赖
cd logic-grpc-service && go mod tidy

# 单元测试（全部）
cd logic-grpc-service && go test ./...

# 特定包测试
cd logic-grpc-service && go test ./ai/...
cd logic-grpc-service && go test ./service/...
cd logic-grpc-service && go test ./migration/...
cd logic-grpc-service && go test ./repository/...

# 带覆盖率
cd logic-grpc-service && go test -coverprofile=coverage.out ./...

# 竞态检测
cd logic-grpc-service && go test -race ./...

# 启动（手动验证）
cd logic-grpc-service && go run main.go    # 监听 :50051
```

### 3.2 web-gin-service

```bash
# 安装依赖
cd web-gin-service && go mod tidy

# 单元测试
cd web-gin-service && go test ./...

# 启动（手动验证，需先启动 logic 服务）
cd web-gin-service && go run main.go       # 监听 :8080
```

### 3.3 Go Vet / 静态检查

```bash
cd logic-grpc-service && go vet ./...
cd web-gin-service && go vet ./...
```

---

## 4. Database Migration

```bash
# migration 自动执行（logic 服务启动时）
# migration 单元测试
cd logic-grpc-service && go test ./migration/...
```

> **待确认**：是否有独立的 migration 命令行入口（如 `go run ./migration/ up/down/status`）。当前 migration 仅在服务启动时自动执行。

---

## 5. Proto 相关

```bash
# Proto 文件位置
ls logic-grpc-service/proto/recruitment.proto
ls web-gin-service/proto/recruitment.proto

# 验证 pb.go 是否最新
diff logic-grpc-service/recruitment/pb/ web-gin-service/recruitment/pb/
```

> **待确认**：Proto 编译命令（`protoc --go_out=...` 或 `buf generate`）。项目未找到 Makefile 或 buf 配置文件。当前 pb.go 可能是手动生成。

---

## 6. Docker Compose

```bash
# 完整部署（用于集成验证，非 CI 自动化）
cd docker
cp .env.example .env   # 编辑填写密钥
docker compose up -d --build

# 仅基础服务
docker compose up -d mysql redis rabbitmq

# 停止
docker compose down
```

---

## 7. 快速验证组合命令

### 仅改前端

```bash
cd hr-frontend && pnpm install && pnpm typecheck && pnpm test && pnpm build
```

### 仅改后端

```bash
cd logic-grpc-service && go vet ./... && go test ./... && go build ./...
```

### 前后端都改

```bash
# 需要按顺序执行，后端先行
cd logic-grpc-service && go vet ./... && go test ./... && echo "Backend OK"
cd web-gin-service && go vet ./... && go test ./... && echo "Gateway OK"
cd hr-frontend && pnpm install && pnpm typecheck && echo "HR Frontend OK"
cd user-frontend && pnpm install && pnpm typecheck && echo "User Frontend OK"
cd interviewer-frontend && pnpm install && pnpm typecheck && echo "Interviewer Frontend OK"
```

---

## 8. 命令可用性备注

| 命令 | hr-frontend | user-frontend | interviewer-frontend | logic-grpc | web-gin |
|------|------------|--------------|---------------------|------------|---------|
| 依赖安装 | `pnpm install` | `pnpm install` | `pnpm install` | `go mod tidy` | `go mod tidy` |
| 类型检查 | `vue-tsc --noEmit` | `vue-tsc --noEmit` | `vue-tsc --noEmit` | `go vet` | `go vet` |
| 单元测试 | `vitest run` | `vitest run` | `vitest run` | `go test ./...` | `go test ./...` |
| 构建 | `vite build` | `vite build` | `vite build` | `go build ./...` | `go build ./...` |
| Lint | **待确认** | **待确认** | **待确认** | `go vet` | `go vet` |
| Proto 生成 | N/A | N/A | N/A | **待确认** | **待确认** |
| Migration CLI | N/A | N/A | N/A | **待确认** | N/A |
