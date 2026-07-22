# Knowledge Audit Contract

## Canonical JSON

Use one temporary JSON document as the sole source of all three deliverables:

```json
{
  "schema_version": 1,
  "audit": {
    "title": "Smart Recruit 项目知识库现状审计报告",
    "repo": "smart-recruit",
    "branch": "main",
    "commit": "0123456789abcdef",
    "working_tree": "clean | dirty",
    "generated_at": "2026-07-22T10:00:00+08:00",
    "scope": "当前工作树中的项目代码与 `.knowledge/`",
    "actual_state_baseline": "WORKTREE",
    "attribution_baseline": "HEAD 0123456",
    "verdict": "UNRELIABLE | DRIFT_DETECTED | CURRENT_WITH_GAPS | CURRENT",
    "overall_risk": "CRITICAL | HIGH | MEDIUM | LOW",
    "limitations": []
  },
  "executive_summary": ["审计结论"],
  "inventory": {
    "active_documents": 1,
    "draft_documents": 0,
    "stale_documents": 0,
    "deprecated_documents": 0,
    "archived_documents": 0,
    "invalid_files": 0
  },
  "coverage": [
    {
      "id": "governance-tooling",
      "dimension": "知识治理与工具链",
      "status": "REVIEWED",
      "evidence": [".knowledge/README.md:1"],
      "limitations": []
    }
  ],
  "documents": [
    {
      "id": "system-overview",
      "path": ".knowledge/architecture/system-overview.md",
      "kind": "architecture",
      "status": "active",
      "last_verified": "2026-07-19",
      "review_after": "2026-10-14",
      "verdict": "UNCHANGED",
      "summary": "关键声明与当前实现一致。",
      "evidence": ["smart-recruit-gateway/router/router.go:20"],
      "finding_ids": []
    }
  ],
  "checks": [
    {
      "id": "CHK-001",
      "name": "知识结构校验",
      "command": "node .knowledge/scripts/validate-knowledge.mjs --root . --json",
      "status": "PASS",
      "result": "零错误",
      "evidence": []
    }
  ],
  "positive_observations": [
    {"title": "现有路由覆盖主要服务", "evidence": [".knowledge/manifest.yaml:1"], "notes": "已验证"}
  ],
  "findings": [
    {
      "id": "KNO-001",
      "title": "候选档案知识文件不符合正式文档格式",
      "type": "INVALID",
      "severity": "MEDIUM",
      "confidence": "CONFIRMED",
      "status": "OPEN",
      "resolution_target": "KNOWLEDGE",
      "document_ids": [],
      "locations": [{"path": ".knowledge/domains/candidate-profile-roadmap.md", "line": 1, "symbol": "frontmatter"}],
      "claim": "domains 下的正式知识文件应包含合法 frontmatter。",
      "actual_state": "文件缺少 opening frontmatter delimiter。",
      "evidence": ["validate-knowledge 返回结构错误"],
      "impact": "知识目录无法通过校验，影响影响分析工具。",
      "attribution": "PRE_EXISTING",
      "recommendation": "确认其生命周期后补齐元数据或迁移到合适位置。",
      "verification": ["知识校验与严格引用检查通过"]
    }
  ],
  "coverage_gaps": [],
  "residual_risks": [],
  "remediation": {"strategy": ["先恢复知识校验，再修复高风险语义漂移。"], "tasks": []}
}
```

All top-level sections shown above are required. Arrays may be empty. Coverage must contain every stable ID from `audit-playbook.md` exactly once. `documents` must contain every active formal document exactly once.

## Verdicts

### Audit verdict

- `UNRELIABLE`: open CRITICAL/HIGH conflict, invalid core governance/tooling, or material active documents remain unreviewed.
- `DRIFT_DETECTED`: confirmed stale/conflicting/invalid knowledge exists without meeting `UNRELIABLE` criteria.
- `CURRENT_WITH_GAPS`: reviewed current knowledge has no confirmed drift, but evidence or coverage gaps remain.
- `CURRENT`: every active document is verified, required coverage is reviewed/applicable, structural checks pass, and no open finding remains.

### Document verdict

- `UNCHANGED`: all material claims reviewed and remain current.
- `STALE`: one or more material claims describe an older implementation.
- `CONFLICT`: a claim directly contradicts higher-authority evidence or another protected decision.
- `CANDIDATE`: useful knowledge or policy needs verification/approval before becoming active truth.
- `INVALID`: the document cannot participate correctly because of structural, lifecycle, or reference defects.

Use the worst material condition as the document verdict. Do not use `UNCHANGED` when review was partial.

### Coverage and checks

Coverage status: `REVIEWED`, `PARTIAL`, `NOT_REVIEWED`, `NOT_APPLICABLE`.

Check status: `PASS`, `FAIL`, `NOT_RUN`, `BLOCKED`, `NOT_APPLICABLE`.

Never summarize `NOT_RUN`, `BLOCKED`, or partial evidence as a pass.

## Findings

Finding types:

- `STALE`
- `CONFLICT`
- `INVALID`
- `COVERAGE_GAP`
- `EVIDENCE_GAP`

Severity:

- `CRITICAL`: likely to cause security/privacy compromise, destructive schema/contract work, or broad production failure when trusted by an Agent.
- `HIGH`: materially misdirects auth, public contract, data ownership, billing, AI/MCP, or cross-service implementation/review.
- `MEDIUM`: causes failed development/operations, meaningful rework, or incomplete review coverage.
- `LOW`: localized navigation, wording, or maintenance drift with bounded impact.

Confidence: `CONFIRMED`, `HIGH`, `MEDIUM`, `LOW`.

Resolution target:

- `KNOWLEDGE`: implementation is authoritative and knowledge should change.
- `SOURCE`: knowledge/decision appears authoritative and implementation may be defective; remediation requires separate source authorization.
- `DECISION_REQUIRED`: evidence cannot safely determine which side should change.

Attribution: `PRE_EXISTING`, `WORKTREE_INTRODUCED`, `MIXED`, `UNKNOWN`.

Finding status: `OPEN`, `RESOLVED`, `ACCEPTED`.

## Contract invariants

- Finding IDs match `KNO-NNN`; check IDs match `CHK-NNN`.
- Every active document has exactly one document entry and verdict.
- Every non-empty document `finding_ids` value references an existing finding that includes that document ID.
- Every open finding maps to at least one remediation task.
- `CURRENT` has no open finding, no structural failure, and no partial/unreviewed mandatory coverage.
- Repository evidence paths are relative POSIX paths; do not embed secrets or personal data.
