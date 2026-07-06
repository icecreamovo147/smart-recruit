# 任务编号：P0-003

## 1. 任务目标

在后端新增 gRPC 接口，支持按会话查询 Agent 工具调用执行轨迹。复用已有 `AIToolTrace` 表和 `ToolTraceRepo.ListBySession` 方法，将其暴露为可供前端调用的 gRPC 接口。

## 2. 对应缺陷

- **缺陷 6**：Agent 执行轨迹后端已落库但前端零展示（后端侧补齐）

## 3. 前置条件

- 无

## 4. 允许修改的文件

- `logic-grpc-service/proto/recruitment.proto`（新增 message + rpc）
- `logic-grpc-service/recruitment/pb/`（重新生成 pb.go）
- `logic-grpc-service/service/ai_service.go`（新增 handler）
- `logic-grpc-service/repository/tool_trace_repo.go`（如需新增查询方法）
- `web-gin-service/proto/recruitment.proto`（同步 proto）
- `web-gin-service/recruitment/pb/`（同步 pb.go）
- `web-gin-service/router/router.go`（注册新路由，如有 HTTP 入口）
- `web-gin-service/handler/`（新增 handler，如有 HTTP 入口）

## 5. 禁止修改的文件

- `hr-frontend/` 下所有文件（前端任务 P0-004）
- `ai/adk_agent.go`、`ai/eino_client.go`
- 已有业务服务文件

## 6. 实现步骤

1. 在 `recruitment.proto` 中新增 message 和方法：
   ```proto
   rpc GetToolTraces(GetToolTracesRequest) returns (GetToolTracesResponse);

   message ToolTraceItem {
     int64 id = 1;
     int64 session_id = 2;
     string tool_name = 3;
     string args_json = 4;       // 脱敏后的工具入参
     string result_content = 5;  // 脱敏后的工具结果
     int64 duration_ms = 6;      // 执行耗时(毫秒)
     string error_msg = 7;       // 错误信息(如有)
     string created_at = 8;
   }
   ```
2. 重新生成 pb.go
3. 在 `ai_service.go` 实现 `GetToolTraces` 方法
4. 查询逻辑：查 `AIToolTrace` 表，按 `session_id` 过滤，按 `created_at` 排序
5. **必须**：在返回结果前对 `args_json` 和 `result_content` 做脱敏处理
6. 同步 proto 到 `web-gin-service`
7. 如需 HTTP 入口，在网关注册路由

## 7. 数据库变更

不涉及（复用已有 `ai_tool_traces` 表）

## 8. 接口变更

- 新增 gRPC method：`AIService.GetToolTraces`
- 新增 message：`GetToolTracesRequest` / `GetToolTracesResponse` / `ToolTraceItem`

## 9. 前端变更

不涉及（P0-004 任务处理）

## 10. 权限与安全要求

- 接口需声明权限：仅 HR 用户可查询，且只能查询自己的会话
- **脱敏要求**：`args_json` 中的手机号、身份证等字段需在返回前脱敏
- **脱敏要求**：`result_content` 中的简历正文需截断或脱敏
- **不记录**：本接口调用本身不需要额外落日志（已有 tool_trace 记录）

## 11. 验收标准

- [ ] `GetToolTraces` gRPC 接口可正常调用
- [ ] 按 `session_id` 返回该会话的所有工具调用记录
- [ ] 返回包含：工具名、入参、结果、耗时、错误信息
- [ ] 入参和结果已做脱敏处理
- [ ] 权限校验：只能查自己的会话
- [ ] `go test ./...` 全部通过
- [ ] proto 在两个服务中同步

## 12. 必须运行的测试命令

```bash
cd logic-grpc-service && go vet ./... && go test ./... && go build ./...
cd web-gin-service && go vet ./... && go test ./... && go build ./...
```

## 13. 完成后必须输出的内容

- 变更摘要
- Proto 变更 diff
- 测试运行结果
- 脱敏逻辑说明

## 14. 回滚方案

- 移除新增 gRPC method（向下兼容，不影响已有方法）
## 15. 完成记录

- 完成日期：2026-06-26
- 提交：`5566205`
- 测试结果：
  - logic-grpc-service: go vet PASS, go test PASS (8 packages), go build PASS
  - web-gin-service: go test PASS (5 packages), go build PASS
  - (go vet 预存问题：test files 中 protobuf struct 值拷贝警告，非本次变更导致)
