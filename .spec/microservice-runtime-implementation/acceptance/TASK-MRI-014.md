# Acceptance - TASK-MRI-014

## 验收标准

- `smart-recruit-offer-service` 可独立构建并启动 gRPC。
- 注册 OfferService。
- Offer 服务通过 gRPC 或事件协作，不新增跨服务写表。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-014`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
