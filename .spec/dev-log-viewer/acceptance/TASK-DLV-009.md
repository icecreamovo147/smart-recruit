# Acceptance - TASK-DLV-009

## TASK Summary

完成跨层、安全、性能有界性、设计状态和知识一致性的最终验收证据。

## SPEC References

- AC-001 至 AC-018
- 全部 NFR、CR、OD、EH、SS
- 第 12 节范围外约束

## SDD References

- 11 测试策略
- 12 迁移风险
- 13 实现边界
- 15 已确认技术决策
- 16 开放问题

## Acceptance Criteria

- 最终报告以表格逐项给出 AC-001 至 AC-018 的证据、命令或人工结果。
- 完整 Go 测试、race 测试、vet、React typecheck/test/build、shell syntax 和 smoke 验证通过。
- 验证 95% 本地追加显示目标、缓冲有界性、慢客户端隔离、重连补发与资源释放；无法稳定自动测量的项目必须记录方法和限制。
- 验证 loopback-only、任意路径不可达、CSP/nosniff/referrer policy、文本安全渲染和零第三方请求。
- 八个设计状态在 1440×900/1366×768 适用视口有记录；不接受页面级双滚动、关键操作溢出或详情覆盖正文。
- knowledge validators、引用检查和 impact detection 通过；每个 routed 文档有固定 verdict。
- 不修改生产实现；发现缺陷时记录所属 TASK 并停止完成判定。

## Required Checks

- `bash .spec/dev-log-viewer/scripts/agent-check.sh TASK-DLV-009`
- `bash .spec/dev-log-viewer/scripts/check-task-scope.sh TASK-DLV-009`
- `node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature .spec/dev-log-viewer --require-pipeline`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree <TASK_BASE_TREE>`

## Manual Verification, if needed

- 按 `design/screens/` 八张基线验证 canonical、compact、selected、reconnecting、paused、high-volume、no-results、no-services。
- 记录浏览器、视口、日期、结果和所有差异。

## Out-of-Scope

新增功能、修改生产代码、Loki/ELK、远程部署、自动脱敏、业务服务控制。
