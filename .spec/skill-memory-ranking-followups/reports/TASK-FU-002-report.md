# TASK Report - TASK-FU-002

## 1. TASK ID

TASK-FU-002 — CI 流水线 protoc 步骤

## 2. Modified File List

- `.github/workflows/ci.yml`（新增 `proto-lint` job）

## 3. Change Summary by File

### `.github/workflows/ci.yml`

新增 `proto-lint` job（在 `frontend` 与 `secret-scan` 之间），4 个步骤：

1. `actions/checkout@v4` + `actions/setup-go@v5`（与 go-test 复用 go 1.25 cache）
2. **Install protoc 25.3**：
   - `apt-get install -y wget unzip`
   - `wget https://github.com/protocolbuffers/protobuf/releases/download/v25.3/protoc-25.3-linux-x86_64.zip`
   - `unzip -o protoc-25.3-linux-x86_64.zip -d /usr/local`
3. **Install protoc-gen-go**：`go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11`
4. **Regenerate recruitment.pb.go**：
   ```bash
   cd logic-grpc-service
   protoc --proto_path=proto \
     --go_out=recruitment/pb \
     --go_opt=paths=source_relative \
     --go_opt=Mrecruitment.proto=recruitment/pb \
     recruitment.proto
   cp recruitment/pb/recruitment.pb.go ../web-gin-service/recruitment/pb/recruitment.pb.go
   ```
5. **Verify pb diff is empty**：
   - 使用 `git diff --quiet` + 显式错误信息（不只是 `git diff --exit-code`）
   - 失败时输出修复步骤提示与 diff stat

### 现有 3 个 job 保持不变
- `go-test`（mysql service + go test）
- `frontend`（matrix: hr-frontend / user-frontend / interviewer-frontend）
- `secret-scan`（gitleaks）

## 4. Scope Check Result

- 实际修改文件：`.github/workflows/ci.yml`，在 `task-scope.json` 的 `allowedFiles` 内。
- `bash .spec/skill-memory-ranking-followups/scripts/agent-check.sh` → 全部通过（含 ci.yml yaml parse 验证）。
- `python3 -c 'import yaml; yaml.safe_load(.github/workflows/ci.yml)'` → 成功。

## 5. SPEC Comparison Result

| SPEC 引用 | 实现情况 |
| --- | --- |
| FR-FU-002 CI 流水线 protoc 步骤 | ✅ `proto-lint` job 完整实现 |
| §7 兼容性 | ✅ 现有 3 个 job 完全保留；新 job 独立运行 |
| §11 AC-FU-002 | ✅ 全部 6 条 AC 满足 |

## 6. SDD Comparison Result

- SDD §3.2 CI protoc 步骤：protoc 25.x 显式安装、protoc-gen-go v1.36.11 显式安装、重生成步骤与本地 TASK-005 完全一致、git diff 检查。
- SDD §8 兼容性：现有 3 个 job 不变。
- SDD §11 测试策略：CI 步骤失败时输出 diff 行号（通过 `git diff --stat`）。

## 7. Acceptance Comparison Result

| AC | 结果 |
| --- | --- |
| AC-001 `.github/workflows/ci.yml` 包含 `proto-lint` job | ✅ |
| AC-002 protoc 25.x 显式安装（wget / unzip） | ✅（PROTOC_VERSION=25.3） |
| AC-003 protoc-gen-go 安装（go install + cache） | ✅（v1.36.11 + go cache 复用） |
| AC-004 重生成两侧 pb 步骤与本地 TASK-005 完全一致 | ✅ |
| AC-005 `git diff --exit-code` 对两侧 pb 必须为空；失败时输出 diff 行号 | ✅（用 `git diff --quiet` + 显式 stat 输出） |
| AC-006 不修改现有 3 个 job | ✅ |

## 8. Test Commands and Results

| 命令 | 结果 |
| --- | --- |
| `python3 -c 'import yaml; yaml.safe_load(.github/workflows/ci.yml)'` | 成功 |
| `grep -E "jobs:\|name: Go Tests\|name: Frontend\|name: Proto Lint\|name: Secret Scan" .github/workflows/ci.yml` | 4 个 job 全部存在 |
| `grep -E "protoc-25\|protoc-gen-go\|diff --quiet" .github/workflows/ci.yml` | 全部匹配 |
| `bash .spec/skill-memory-ranking-followups/scripts/agent-check.sh` | 全部通过 |

## 9. 已知风险与首次运行状态

**首次运行 CI 会失败**（这是预期行为，不是 bug）：

- 原因：当前 git HEAD 提交的 `recruitment.pb.go` 是用更早版本的 protoc-gen-go 生成的（约 v1.30 时代），而 CI 步骤会使用 protoc-gen-go v1.36.11 重生成。
- 影响：CI step 5（Verify pb diff is empty）会触发"diff found" 错误，输出修复提示与 diff stat。
- 修复方式：开发者本地重生成 + commit 两侧 pb 文件；之后 CI 会一直 pass 直到下次 proto 变更。
- 这是有意的设计：CI 步骤的职责是"防止 .proto 与 .pb.go 长期不一致"；当前不一致是历史遗留，CI 步骤"提醒"开发者修复。

**未来价值**：
- 开发者修改 .proto 后，CI 会立即提醒需要重生成 + commit。
- 防止 proto 字段被无意修改后未同步 pb 文件。

## 10. 后续建议

- **首次 commit**：开发者（用户）需要在本地跑一次 `protoc` 重生成 + 提交 .pb.go，使 CI 通过。
- 后续 PR 修改 .proto 时，CI 会自动检查 .pb.go 一致性。
- 后续可考虑用 `buf` 替代 protoc + 手工 path mapping（SPEC §14 备选方案），但需要单独流程。

## 11. Whether the Next TASK Can Start

✅ TASK-FU-003 可以开始。CI 配置已完成；前端 UI 升级与 CI 独立。
