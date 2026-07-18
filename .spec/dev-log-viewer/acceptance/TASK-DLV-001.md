# Acceptance - TASK-DLV-001

## TASK Summary

建立根目录 `dev-log-viewer/` 的 Go + React 工程骨架、已批准依赖和 workspace 基线，不实现日志业务。

## SPEC References

- G-005、FR-025、FR-026
- CR-001、CR-008、CR-009
- AC-013、AC-015
- D-001、D-002、D-003、D-010、D-011

## SDD References

- 1.3 前端和工作区
- 3.1 模块位置
- 7 配置设计
- 13.1 允许涉及的区域
- 15 已确认技术决策

## Acceptance Criteria

- `dev-log-viewer/` 同时具备独立 `go.mod` 和前端 `package.json`。
- `pnpm-workspace.yaml` 只新增 `dev-log-viewer` 成员；现有成员保持不变。
- manifest 仅包含 SPEC 已批准依赖，锁文件无无关 package 升级。
- React/Vite/Tailwind 占位入口可 typecheck、测试和生产构建。
- 不使用 CDN、网络字体、`any` 绕过或现有 Vue 应用代码。

## Required Checks

- `pnpm install --lockfile-only` 或仓库等价依赖安装命令，记录真实结果。
- `pnpm --filter dev-log-viewer typecheck`
- `pnpm --filter dev-log-viewer test`
- `pnpm --filter dev-log-viewer build`
- `cd dev-log-viewer && go test ./...`
- Harness scope 与 agent-check。

## Manual Verification, if needed

- 审查 `pnpm-lock.yaml` diff，确认没有非必要全局升级。

## Out-of-Scope

日志读取、服务 API、SSE、正式 UI、启停脚本和知识库修改。
