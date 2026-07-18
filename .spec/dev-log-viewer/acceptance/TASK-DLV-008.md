# Acceptance - TASK-DLV-008

## TASK Summary

完成内嵌前端的单进程运行、显式启停 target、模块 README 和知识事实更新。

## SPEC References

- FR-025 至 FR-028
- CR-001 至 CR-009
- OD-003、SS-001、SS-006、SS-007
- AC-012、AC-013、AC-015、AC-018
- D-001 至 D-003、D-006、D-010、D-011

## SDD References

- 3.1 模块位置
- 5.4 静态资源
- 7 配置设计
- 8 兼容性策略
- 10 可观测性
- 13 实现边界

## Acceptance Criteria

- 明确的构建命令先产出 `web/dist`，生产 Go build 将该产物嵌入同一二进制；开发路径不依赖陈旧 dist。
- 单进程同源提供 UI、API 和 SSE，并设置 SDD 规定的安全响应头。
- `./start-dev.sh logs` 与 `log-viewer` 启动查看器并写入独立 PID/日志；stop 对应生效。
- 无参数与 `all` 的既有 12 服务展开结果不变，查看器退出不影响它们。
- README 记录启动、构建、端口、配置、容量、安全提示和故障排查。
- knowledge manifest 能路由 `dev-log-viewer/**`、start/stop 变更；system-overview 与 local-development 只写已验证事实且 source_refs 对齐。

## Required Checks

- `pnpm --filter dev-log-viewer build`
- 按 README 执行生产 Go build，然后 `cd dev-log-viewer && go test ./... && go vet ./...`
- `bash -n start-dev.sh stop-dev.sh`
- 启停脚本显式 target smoke test，不启动/停止无关服务。
- `node .knowledge/scripts/knowledge-validator.test.mjs`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree <TASK_BASE_TREE>`
- Harness scope 与 agent-check。

## Manual Verification, if needed

- 浏览器打开 `http://127.0.0.1:8090`，确认静态资源和 API 同源且无第三方网络请求。

## Out-of-Scope

加入默认/`all`、根 README、根 docs、Docker/CI、业务服务配置。
