# Development Agent Knowledge Base SDD

## 1. Existing Architecture Summary

### 1.1 Canonical Agent 控制面

- `AGENTS.md` 定义仓库持久规则、架构边界、测试规范以及 SPEC + SDD + Harness 顺序。
- `.agents/skills/spec-harness/SKILL.md` 定义 feature 初始化、单 TASK 实施、Review、修复、scope 和 evidence 契约。
- `.agents/skills/harness-pipeline/SKILL.md` 仅在用户明确调用时执行串行多 TASK 编排。
- `.spec/<feature-name>/` 保存可执行需求、设计、TASK、Acceptance、scope、报告和运行证据。
- `pipeline-state.json` 是 pipeline 运行时状态来源；`task-scope.json.tasks[*].status` 只是生成时初始状态。

### 1.2 当前知识来源

- `README.md` 提供项目能力、架构和本地运行简介。
- `.ai-guides/` 是 Git 忽略的历史指南和旧交付材料，其 README 已声明不是直接实现输入。
- 代码、Proto、`db.sql`、测试和配置示例是实现事实来源。
- `.spec/skill-memory-ranking/`、`.spec/semantic-retrieval-score-fixes/` 等包含 Agent Skill、Memory 和 Embedding 的有效或历史 feature 设计，可作为核验线索。
- `CLAUDE.md` 与 `.claude/CLAUDE.md` 已是 canonical 控制面的薄适配器，不应产生 provider 专属知识副本。

### 1.3 项目结构与工具约束

- 三个 Vue 3 前端通过 `pnpm-workspace.yaml` 组织，但仓库根目录没有 `package.json`。
- 两个 Go 服务各自拥有 `go.mod`。
- Harness 共享校验器使用 Node `.mjs` 和 Git，不依赖第三方包。
- 当前根目录没有 `.gitattributes`，也未发现现有 `.github` 工作流。
- `.gitignore` 当前未忽略 `.knowledge`，因此正式知识默认可被 Git 跟踪。

## 2. Problem Analysis

### 2.1 信息分散且角色不同

仓库规则、当前 feature 契约、源代码、历史方案和运行时 Memory 承担不同角色。若不建立明确边界，新增知识库容易复制事实、覆盖 SPEC 或把历史结论误当当前实现。

### 2.2 自由判断依赖模型能力

如果仅要求 Agent “发现高价值知识时维护”，弱模型容易漏掉跨模块影响、公共契约变化和失效 Runbook。方案必须让机械规则决定检查范围，让 Agent 在固定 verdict 中基于证据判断结果。

### 2.3 多设备和多分支风险

绝对路径、大小写文件名、Shell 差异、本地缓存和全局频繁重写文件会导致 macOS、Linux、Windows/WSL2 和并行分支之间的不一致。

### 2.4 知识可信度与提示注入

Agent 生成内容、外部材料和历史指南可能未经验证。若普通 Markdown 被当成指令，可能绕过 canonical 控制面或引入敏感信息，因此正式知识必须可追溯并受分级审批。

## 3. Proposed Design

### 3.1 总体结构

在根目录建立 `.knowledge/`，将内容分为治理入口、正式知识、候选/归档、模板/schema、工具和本地状态：

```text
.knowledge/
  README.md
  INDEX.md
  manifest.yaml
  .gitignore
  .gitattributes
  architecture/
  domains/
  decisions/
  runbooks/
  pitfalls/
  inbox/
  archive/
  templates/
  schemas/
  scripts/
  generated/   # ignored
  .local/      # ignored
```

### 3.2 元数据单一来源

- 每篇文档 frontmatter 是该文档 ID、kind、status、owner、tags、路径和来源的权威元数据。
- `manifest.yaml` 只保存全局政策、路径 routes 和 triggers，不复制单篇可变元数据。
- `INDEX.md` 是稳定的人类路由，不维护动态统计或完整文件清单。
- 完整 catalog 从 frontmatter 确定性生成到 ignored 的 `generated/`。

### 3.3 信任分区

- `active` 正式知识可用于导航，但关键结论仍需回查来源。
- `draft` Inbox、`stale`、`deprecated`、`archived` 默认不进入实现上下文。
- 普通知识正文无指令权；`AGENTS.md`、当前 `.spec` 与治理 README 始终优先。

### 3.4 维护分级

- L1：路径、命令、入口、链接、已验证步骤，可直接更新。
- L2：行为、职责、数据流，需要证据和 Review。
- L3：决策、安全、权限、公共契约，只能候选或 proposed ADR，并要求人工确认。

## 4. Data Structure Changes

本功能不修改数据库或业务模型，只新增文件结构与 JSON/YAML schema。

### 4.1 Frontmatter schema

概念结构：

```yaml
schema_version: 1
id: semantic-retrieval
title: Agent Skill 与 Memory 语义召回
kind: architecture
status: active
owners: [agent-platform]
tags: [agent, skill, memory, embedding]
applies_to:
  - logic-grpc-service/service/agent_skill_selector.go
source_refs:
  - .spec/skill-memory-ranking/skill-memory-ranking-SDD.md
last_verified: 2026-07-10
review_after: 2026-10-10
```

ADR 可增加：

```yaml
decision_status: proposed | accepted | rejected | superseded
supersedes: []
superseded_by: null
```

### 4.2 Manifest schema

```yaml
schema_version: 1
policies:
  default_review_days: 90
  allowed_statuses: []
  direct_update_kinds: []
  approval_required_kinds: []
routes:
  - match: []
    documents: []
    triggers: []
global_triggers: []
```

### 4.3 Knowledge impact schema

Harness evidence 中采用可选、向后兼容字段：

```json
{
  "knowledgeImpact": {
    "result": "update_required",
    "triggeredBy": ["covered-path-changed"],
    "reviewResults": [
      {
        "document": ".knowledge/architecture/semantic-retrieval.md",
        "verdict": "UPDATED",
        "evidence": ["logic-grpc-service/service/embedding_service.go"]
      }
    ],
    "coverageGap": false,
    "validationExitCode": 0
  }
}
```

历史 evidence 缺少该字段仍保持可读。

## 5. API and Interface Changes

### 5.1 CLI 接口

```text
node .knowledge/scripts/validate-knowledge.mjs --root <repo> [--json]
node .knowledge/scripts/check-references.mjs --root <repo> [--json]
node .knowledge/scripts/detect-impact.mjs --root <repo> --base-tree <git-tree> [--json]
node .knowledge/scripts/generate-catalog.mjs --root <repo> [--output <path>]
```

错误使用返回退出码 2；校验或影响契约失败返回 1；成功返回 0。

### 5.2 Harness 接口

共享 spec-harness 在保持旧 feature 兼容的前提下：

- 允许 TASK/acceptance 声明知识复核和修改范围。
- 报告模板增加 Knowledge Impact 段落。
- evidence 校验器在新 feature 声明知识要求时校验 `knowledgeImpact`，不追溯强制历史 feature。
- scope 仍只依据 `allowedFiles`、`forbiddenFiles` 和可靠 Git baseline。

### 5.3 无业务 API 变化

不修改 HTTP、gRPC、Proto、数据库或前端接口。

## 6. Algorithm or Workflow Changes

### 6.1 任务开始路由

```text
任务目标和预计文件范围
  -> 读取 AGENTS 与当前 .spec
  -> 读取 .knowledge/README
  -> manifest route 匹配
  -> INDEX 领域补充
  -> 过滤 active 正式知识
  -> 回查关键 source_refs
```

### 6.2 TASK 结束影响检测

```text
可靠 TASK base tree
  -> git diff + untracked 文件
  -> 路径归一化为 POSIX 仓库相对路径
  -> route 匹配
  -> global trigger 候选检测
  -> 文档复核清单
  -> coverage gap
  -> Agent 逐篇固定 verdict
  -> scope 内更新 / Inbox 候选 / 报告债务
  -> 校验与 evidence
```

工具不裁决架构语义，只提供确定性输入。Agent 或 Reviewer 负责解释 `service-boundary-changed` 等语义触发器是否成立并提供证据。

### 6.3 路径匹配

- 所有路径在匹配前将 `\` 转为 `/`。
- 禁止以 `/`、盘符或用户 home 开头的知识路径。
- glob 语义与共享 Harness 校验器保持一致：`*` 不跨目录，`**` 可跨目录。
- 输出按规范化路径和文档 ID 排序。

### 6.4 Coverage gap

新增顶层模块、公共接口、Schema、核心 service 或配置入口未命中任何 route 时，工具输出 gap candidate。它只要求记录或提出候选，不自动创建正式知识。

## 7. Configuration Design

### 7.1 本地 Git 规则

`.knowledge/.gitignore`：

```gitignore
.local/
generated/
```

`.knowledge/.gitattributes` 在目录局部规范 Markdown、YAML、JSON、MJS 的 LF 换行，避免修改根级 `.gitattributes`。

### 7.2 Manifest 配置

V1 默认复核周期 90 天。routes 必须引用已存在知识 ID；owner 名称在初始知识落地前确认。

### 7.3 CI 配置

CI 平台尚未从仓库确定。TASK-007 在人工确认后创建最窄范围的 CI 集成；若不使用 GitHub Actions，则保留稳定 CLI 并记录外部流水线接入方式。

## 8. Compatibility Strategy

1. `.knowledge` 是 canonical 控制面的下游导航层，不改变现有权威关系。
2. Knowledge impact 在 Harness/evidence 中为新 feature 可选能力，历史 feature 不批量迁移。
3. 不修改业务代码、共享类型、Proto 或数据库。
4. Provider adapters 继续只引用 `AGENTS.md` 和 canonical skills。
5. 工具使用 Node 标准库，避免根 package manifest。
6. 知识文件和脚本使用仓库相对路径，适配不同设备目录。

## 9. Error Handling and Fallback Design

| 场景 | 行为 |
|---|---|
| 无可靠 base tree | `detect-impact` 退出 2，不输出通过结论 |
| manifest 无效 | 校验退出 1，列出字段或 route 错误 |
| frontmatter 无效 | 校验退出 1，定位文件和字段 |
| source ref 不存在 | 正式知识失败；候选知识报告 warning 或按 schema 失败 |
| route 引用未知 ID | 校验失败 |
| 知识与权威来源冲突 | Agent verdict 为 `CONFLICT`，停止自动提升 |
| TASK scope 不允许更新 | verdict 为 `STALE`/`CANDIDATE`，只记录债务 |
| 新路径未覆盖 | 输出 `coverage_gap` |
| catalog 目录不存在 | 工具在 ignored `generated/` 内创建；不得修改正式知识 |
| Node/Git 不存在 | 明确失败，不跳过并宣称成功 |

## 10. Observability and Debug Output Design

人类输出示例：

```text
base_tree: <sha>
changed_files: 4
matched_routes: 2
review_documents:
  - semantic-retrieval
coverage_gaps: 0
result: REVIEW_REQUIRED
```

JSON 输出包含 schema version、base tree、changed files、routes、triggers、documents、gaps 和建议模板。输出不得包含知识全文、Secret 或未脱敏业务数据。

## 11. Testing Strategy

### 11.1 单元测试

使用 Node 标准库和临时目录覆盖：

- Frontmatter 解析与必需字段。
- ID 唯一性。
- kind/status 枚举。
- 相对路径与 Windows 路径归一化。
- 绝对路径拒绝。
- route/glob 匹配。
- 稳定排序与 JSON 输出。
- source refs、INDEX、ADR 关系。
- Inbox/Archive 状态。
- tracked/untracked diff 和 invalid baseline。
- coverage gap。

### 11.2 Harness 回归

- 运行 `.agents/skills/spec-harness/scripts/validator.test.mjs`。
- 验证至少一个历史 feature 仍为 current 或 legacy-compatible，而不是因缺少 knowledgeImpact 变为 unsupported。
- 新 feature 的报告/evidence 缺失 required knowledgeImpact 时失败。

### 11.3 内容验收

- 首批知识逐篇核对 source refs。
- 使用代表性的 Agent Skill、Memory、Proto、HR 管理页路径验证路由。
- 检查仓库中 `.knowledge` 文件无绝对路径、敏感数据和 provider 专属规则副本。

### 11.4 跨平台

- 本地至少验证 macOS/Linux 兼容命令。
- CI 平台允许时增加 Windows 或 WSL2 路径用例；路径归一化必须由单测覆盖，不依赖实际 Windows runner 才能验收核心逻辑。

## 12. Migration Risks

1. 历史材料可能已过时；首批知识必须重新核验，不能机械迁移。
2. routes 过宽会产生大量误报，过窄会漏检；通过试运行调整。
3. 单一 INDEX 或 manifest 频繁重写会产生分支冲突；保持稳定路由和确定排序。
4. AGENTS 与 Harness 集成属于共享控制面变更，必须独立 TASK、人工确认和回归。
5. 当前 CI 平台未知，不能在 init-feature 阶段假设并修改 `.github`。
6. YAML 解析不应引入依赖；V1 frontmatter/manifest 子集需要明确限制并由测试保护。

## 13. Implementation Boundaries

### TASK-001：基础契约

只创建 `.knowledge` 治理文档、manifest、schemas、templates、局部 Git 规则和目录说明。

### TASK-002：校验与影响检测工具

只创建 `.knowledge/scripts/` 和测试；不得修改 package manifests。

### TASK-003：架构与领域知识

只创建首批 architecture/domain 文档，并核验当前代码和有效 SPEC。

### TASK-004：Runbook 与 Pitfall

只创建首批 runbook/pitfall 文档，并核验命令、路径和行为。

### TASK-005：AGENTS 接入

修改共享 `AGENTS.md` 前必须人工确认；不得同步改 Provider adapter，除非发现冲突并另行扩展 scope。

### TASK-006：共享 Harness 接入

修改 `.agents/skills/spec-harness/` 前必须人工确认；保持历史 feature 兼容，不修改 pipeline 状态语义。

### TASK-007：CI 接入

确认 CI 平台后才允许修改 CI 配置；不修改部署和业务配置。

### TASK-008：最终审计

只写 feature 报告/evidence；发现问题返回所属 TASK，不直接修共享文件。

## 14. Alternatives Considered

### 14.1 由 Agent 自由决定何时维护

拒绝。弱模型漏判风险高，改为所有非简单 TASK 强制影响检查。

### 14.2 每次加载全部知识

拒绝。上下文成本和误导风险随规模增长，采用 route + INDEX 按需读取。

### 14.3 V1 使用 Embedding

拒绝。当前规模使用 frontmatter、glob 和文本检索足够，且避免运行时系统耦合。

### 14.4 全量迁移 `.ai-guides`

拒绝。历史材料不具备当前权威性，只作为线索逐篇核验。

### 14.5 根级 `.gitignore` 与 `.gitattributes`

默认不采用。使用 `.knowledge` 内局部文件减少全局配置影响；若 Git 行为验证不满足再提出独立变更。

### 14.6 每个 Agent Provider 维护独立知识

拒绝。所有 Provider 通过 canonical 控制面读取同一 `.knowledge`。

## 15. Assumptions Requiring Confirmation

1. 默认复核周期 90 天。
2. owner 标识在 TASK-003 前由用户确认或采用已在 SPEC 标注的稳定模块名。
3. TASK-005、TASK-006、TASK-007 均在实施前单独取得人工确认。
4. CI 平台未确认前，不创建实际工作流文件。
5. YAML 支持范围限定为本功能模板使用的简单 mapping/list/scalar 子集，避免引入 parser 依赖。

## 16. Open Questions

1. 最终 owner taxonomy。
2. CI 平台与所需 runner 矩阵。
3. 不同 kind 是否采用 30/90/180 天差异化复核周期。
4. TASK-008 的真实试运行样本。
