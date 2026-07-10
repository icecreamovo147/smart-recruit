# Legacy Inventory - agent-harness-unification

This inventory is read-only registration. It does not cancel, complete, activate, or implement legacy work. `Current Code Verified` is `No` unless a separate migration feature explicitly re-checks the current code.

## Legacy Agent Harness Pending Tasks

| Source | Legacy ID | Historical Status | Current Code Verified | Equivalent Spec | Migration Decision | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| `docs/agent-harness/tasks/P1-008-调试面板-后端.md` | P1-008 | pending | No | Not found | migrate-on-demand | Create a new `.spec` feature after current-code analysis and user confirmation. |
| `docs/agent-harness/tasks/P1-009-调试面板-前端.md` | P1-009 | pending | No | Not found | migrate-on-demand | Create a new `.spec` feature after current-code analysis and user confirmation. |
| `docs/agent-harness/tasks/P1-010-输入区增强.md` | P1-010 | pending | No | Not found | migrate-on-demand | Create a new `.spec` feature after current-code analysis and user confirmation. |
| `docs/agent-harness/tasks/P1-011-脱敏中间件与二次确认.md` | P1-011 | pending | No | Not found | migrate-on-demand | Security-sensitive; migration must re-check current auth, audit, and masking behavior. |
| `docs/agent-harness/tasks/P1-012-数据源管理-后端.md` | P1-012 | pending | No | Not found | migrate-on-demand | Create a new `.spec` feature after current-code analysis and user confirmation. |
| `docs/agent-harness/tasks/P1-013-数据源管理-前端.md` | P1-013 | pending | No | Not found | migrate-on-demand | Create a new `.spec` feature after current-code analysis and user confirmation. |
| `docs/agent-harness/tasks/P1-014-知识库RAG-后端.md` | P1-014 | pending | No | Not found | migrate-on-demand | RAG/provider design must be revalidated before implementation. |
| `docs/agent-harness/tasks/P1-015-知识库RAG-前端.md` | P1-015 | pending | No | Not found | migrate-on-demand | Depends on a migrated backend/data contract. |
| `docs/agent-harness/tasks/P2-001-Skills管理.md` | P2-001 | pending | No | Not found | migrate-on-demand | Must not reuse old Agent task assumptions without current-code analysis. |
| `docs/agent-harness/tasks/P2-002-对话结构化结果展示.md` | P2-002 | pending | No | Not found | migrate-on-demand | Requires current UI/API contract analysis before a SPEC. |
| `docs/agent-harness/tasks/P3-001-面试官AI助手.md` | P3-001 | pending | No | Not found | migrate-on-demand | Large cross-surface feature; split only after canonical SPEC/SDD. |

## Legacy AI Guide Phase Contracts

| Source | Legacy ID | Historical Status | Current Code Verified | Equivalent Spec | Migration Decision | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| `.ai-guides/candidate-match-evaluation-algorithm` | candidate-match-evaluation-algorithm | legacy phase contract; delivery: no; acceptance: no | No | Not found | migrate-on-demand | Contains constitution/spec/plan/tasks; must be revalidated before implementation. |
| `.ai-guides/hr-ai-context-usage` | hr-ai-context-usage | legacy phase contract; delivery: no; acceptance: no | No | Not found | migrate-on-demand | Contains constitution/spec/plan/tasks; must be revalidated before implementation. |
| `.ai-guides/recruitment-mainline-phase-1-pipeline-state` | recruitment-mainline-phase-1-pipeline-state | legacy phase contract; delivery: yes; acceptance: yes | No | Not found | archive-reference | Delivery and acceptance exist; retain as historical evidence unless new work is requested. |
| `.ai-guides/recruitment-mainline-phase-2-interview-workflow` | recruitment-mainline-phase-2-interview-workflow | legacy phase contract; delivery: yes; acceptance: yes | No | Not found | archive-reference | Delivery and acceptance exist; retain as historical evidence unless new work is requested. |
| `.ai-guides/recruitment-mainline-phase-3-offer-onboarding` | recruitment-mainline-phase-3-offer-onboarding | legacy phase contract; delivery: yes; acceptance: yes | No | Not found | archive-reference | Delivery and acceptance exist; retain as historical evidence unless new work is requested. |
| `.ai-guides/recruitment-mainline-phase-4-candidate-collaboration` | recruitment-mainline-phase-4-candidate-collaboration | legacy phase contract; delivery: yes; acceptance: yes | No | Not found | archive-reference | Delivery and acceptance exist; retain as historical evidence unless new work is requested. |
| `.ai-guides/recruitment-mainline-phase-5-rbac-ops-scope` | recruitment-mainline-phase-5-rbac-ops-scope | legacy phase contract; delivery: yes; acceptance: yes | No | Not found | archive-reference | Delivery and acceptance exist; retain as historical evidence unless new work is requested. |
| `.ai-guides/recruitment-mainline-phase-6-analytics-ai-audit` | recruitment-mainline-phase-6-analytics-ai-audit | legacy phase contract; delivery: yes; acceptance: yes | No | Not found | archive-reference | Delivery and acceptance exist; retain as historical evidence unless new work is requested. |
| `.ai-guides/recruitment-mainline-roadmap` | recruitment-mainline-roadmap | legacy phase contract; delivery: no; acceptance: no | No | Not found | archive-reference | Roadmap-style material; use as planning reference only. |
| `.ai-guides/role-permission-redesign` | role-permission-redesign | legacy phase contract; delivery: yes; acceptance: yes | No | Not found | archive-reference | Delivery and acceptance exist; retain as historical evidence unless new work is requested. |

## Non-Current `.spec` Feature Requiring Separate Repair

| Source | Legacy ID | Historical Status | Current Code Verified | Equivalent Spec | Migration Decision | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| `.spec/semantic-retrieval-score-fixes` | semantic-retrieval-score-fixes | pending `.spec` with unsupported Harness shape | No | Existing but unsupported | separate-repair | Do not rewrite in this feature. Create a dedicated Harness repair feature before implementation can resume. |
