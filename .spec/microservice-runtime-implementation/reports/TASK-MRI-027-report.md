# TASK-MRI-027 Report

## TASK ID

TASK-MRI-027

## 修改文件列表

- `.spec/microservice-runtime-implementation/pipeline-state.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-027-evidence.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-027-report.md`
- `.spec/microservice-runtime-implementation/scripts/agent-check.sh`
- `scripts/check-redis-prefixes.mjs`
- `scripts/redis-prefix-exceptions.json`
- `smart-recruit-platform-go/rediskey/builder.go`
- `smart-recruit-platform-go/rediskey/builder_test.go`

## 每个文件的变更摘要

- `.spec/microservice-runtime-implementation/pipeline-state.json`: 记录 `TASK-MRI-027` 的基线、通过状态和 evidence 路径，并推进当前任务到 `TASK-MRI-028`。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-027-evidence.json`: 记录本 TASK 的机器可读检查、自审、scope 与知识影响证据。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-027-report.md`: 记录本 TASK 的人工可读验收报告。
- `.spec/microservice-runtime-implementation/scripts/agent-check.sh`: 接入 `node scripts/check-redis-prefixes.mjs`，使 Redis prefix 检查成为后续 TASK 的通用门禁。
- `scripts/check-redis-prefixes.mjs`: 新增 Redis key 静态检查，扫描 go-redis client/script/pubsub 调用，要求新增 key 使用 platform helper 或显式登记例外。
- `scripts/redis-prefix-exceptions.json`: 记录当前 web-gin 与 monolith fallback 中未迁移的 legacy Redis key 例外。
- `smart-recruit-platform-go/rediskey/builder.go`: 新增服务级 Redis key/channel builder，强制服务 prefix 与非空 key part。
- `smart-recruit-platform-go/rediskey/builder_test.go`: 覆盖 prefix 生成、channel 生成、空服务拒绝、空 part 拒绝和自定义分隔符。

## Scope check 结果

通过。命令：

```bash
TASK_BASE_TREE=78541b3daf8860d1fe9aead9056a453faa5b98a7 bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-027
```

初次 scope check 发现 `go.work.sum` 越界；已删除该 Go 测试生成物，最终 scope check 通过，所有保留变更均匹配允许范围。

## SPEC 对照结果

通过。共享 Redis 现在具备平台级服务 prefix helper 和可执行门禁；未提交 secrets、未改变数据库、未新增跨服务直接写表。

## SDD 对照结果

通过。运行时平台层新增可复用 Redis key builder，agent-check 会阻止新增裸 Redis key；legacy gateway/monolith fallback 访问以例外清单追踪。

## Acceptance 对照结果

通过。Platform helper 强制服务 prefix；负向 `/tmp` fixture 验证检查脚本能发现新增无 prefix Redis key；Gateway 与服务缓存使用方的未迁移项已记录为可追踪例外。

## 测试命令和结果

- `cd smart-recruit-platform-go && go test ./rediskey`: passed
- `cd smart-recruit-platform-go && go test ./...`: passed
- `node scripts/check-redis-prefixes.mjs`: passed
- `REDIS_PREFIX_EXTRA_ROOTS=/tmp/... node scripts/check-redis-prefixes.mjs (negative bare-key fixture)`: passed
- `git diff --name-only`: passed
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 78541b3daf8860d1fe9aead9056a453faa5b98a7 --json`: passed
- `TASK_BASE_TREE=78541b3daf8860d1fe9aead9056a453faa5b98a7 bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-027`: passed
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`: passed
- `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/microservice-runtime-implementation/reports/TASK-MRI-027-evidence.json`: passed

## Self-review

- 是否真实落地，而不是只写文档：通过，已新增平台 helper、测试、检查脚本和 agent-check 门禁。
- 是否越过 TASK scope：通过，`go.work.sum` 生成物已移除，最终 scope check 无越界或 forbidden 文件。
- 是否破坏 HTTP/protobuf 兼容：通过，未修改 HTTP routes、handlers、protobuf 或生成代码。
- 是否提交 secrets 或真实 `.env`：通过，未创建或修改 env/secrets。
- 是否违反单 MySQL 约束：通过，本 TASK 仅涉及 Redis key 隔离，未修改 MySQL schema、实例或访问方式。
- 是否缺少测试、scope check、agent-check 或 evidence validation：通过，平台测试、Redis prefix 正向/负向检查、scope check、agent-check 和 evidence validation 均已通过。

verdict: 通过

## Knowledge Impact

`update_required`。已按路由审阅 `.knowledge/architecture/system-overview.md` 和 `.knowledge/runbooks/local-development.md`，结论均为 `UNCHANGED`；本 TASK 新增 Redis prefix 验证门禁，不改变公开 API、启动顺序或服务拓扑。

## 风险

- Legacy Redis key 使用因本 TASK scope 不能修改 `web-gin-service/**` 和 `logic-grpc-service/**`，已登记为例外；后续 TASK 若允许触达对应文件，应逐步迁移并删除例外。

## 下一 TASK 是否可以开始

可以，在 evidence validator 通过、pipeline-state 记录完成并创建本地 commit 后进入 TASK-MRI-028。
