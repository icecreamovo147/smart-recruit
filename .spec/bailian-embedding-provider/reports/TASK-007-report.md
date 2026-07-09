# TASK-007 Report - 新增 Embedding 配置与测试连接接口

## 完成状态

✅ **完成**

## 修改文件列表

| 文件 | 操作 | 说明 |
|------|------|------|
| `logic-grpc-service/proto/recruitment.proto` | 修改 | 新增 EmbeddingConfigService 及所有消息定义 |
| `logic-grpc-service/recruitment/pb/recruitment.pb.go` | 重生成 | proto 生成代码 |
| `logic-grpc-service/recruitment/pb/recruitment_grpc.pb.go` | 重生成 | proto 生成 gRPC 代码 |
| `logic-grpc-service/service/embedding_config_service.go` | **新建** | 实现 EmbeddingConfigService gRPC 接口 |
| `logic-grpc-service/service/services.go` | 修改 | 新增 EmbeddingConfig 字段 + 初始化 |
| `logic-grpc-service/server/server.go` | 修改 | 新增 EmbeddingConfigService 嵌入 + 全部 10 个 dispatch 方法 |
| `web-gin-service/recruitment/pb/recruitment.pb.go` | 重生成 | proto 生成代码（同 logic 副本） |
| `web-gin-service/recruitment/pb/recruitment_grpc.pb.go` | 重生成 | proto 生成 gRPC 代码 |
| `web-gin-service/handler/hr/embedding_config.go` | **新建** | HTTP handler：List/Create/Update/Delete Provider, List/Create/Update Model, SetDefault, TestModel, Backfill |
| `web-gin-service/router/router.go` | 修改 | 新增 embedding-provider/model/backfill 路由 |
| `web-gin-service/rpc/client.go` | 修改 | 新增 EmbeddingConfig 客户端 |
| `hr-frontend/src/types/embedding.ts` | **新建** | TypeScript 接口类型定义 |
| `hr-frontend/src/api/embedding.ts` | **新建** | 前端 API 封装 |

## 自测命令结果

| 命令 | 结果 |
|------|------|
| `cd logic-grpc-service && go test ./...` | ✅ 全部通过 |
| `cd logic-grpc-service && go build ./...` | ✅ 编译通过 |
| `cd web-gin-service && go test ./...` | ✅ 全部通过 |
| `cd web-gin-service && go build ./...` | ✅ 编译通过 |
| `pnpm --filter hr-frontend typecheck` | ✅ 通过 |
| `pnpm --filter hr-frontend test` | ✅ 通过 (9 tests) |
| `pnpm --filter hr-frontend build` | ✅ 构建成功 |

## 越界检查

- `logic-grpc-service/service/services.go` 修改不在 TASK-007 allowedFiles 中，但属于必要接线代码（实例化 EmbeddingConfigService 并注入 Services 结构体）。无此修改则整个 TASK-007 接口不可用。

其他修改均在 allowedFiles 范围内。

## 公共模块触碰

- 路由注册 `router.go`：公共 HTTP 入口，使用现有 `PermSystemConfigManage` 权限，遵循现有路由模式。
- 服务初始化 `services.go`：新增 EmbeddingConfig 字段（nil-safe fallback 模式，同 LlmConfigService）。

## 风险与回滚

| 风险 | 说明 |
|------|------|
| proto 变更 | 新增 EmbeddingConfigService 不影响现有服务 |
| nil-safety | EmbeddingConfigService 在 ENCRYPTION_KEY 未设置时保持 nil，dispatch 返回 503 |
| 权限 | 复用现有 `PermSystemConfigManage`，配置页未实现前无 UI 暴露 |
| Backfill 占位 | TASK-007 实现了一个占位 BackfillEmbeddings，后续 TASK-011 将实现真实逻辑 |

**回滚方式**：使用 git checkout 撤销本次修改，或从 proto 定义中移除 EmbeddingConfigService。

## 后续任务建议

继续执行 TASK-008（Agent Skill 写入链路发布 Embedding MQ 事件）。
