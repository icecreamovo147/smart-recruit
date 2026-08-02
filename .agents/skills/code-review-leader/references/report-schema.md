# Code Review Leader Report Schema

Use this reference when shaping the JSON consumed by `scripts/render_review_report.mjs`.

## Top-Level Fields

- `title`: Optional. Defaults to `Code Review Leader — 变更可视化报告`.
- `repo`: Repository display name.
- `branch`: Current branch.
- `generated_at`: Human-readable report time.
- `scope`: Review range, such as `working tree vs HEAD` or `main...feature`.
- `scope_type`: `branch`, `working-tree`, `commit`, or `range`.
- `reviewed_commit`: Commit hash when reviewing one commit.
- `verdict`: Merge recommendation.
- `risk_level`: Overall risk label.
- `summary`: Array of summary cards.
- `sections`: Array of collapsible report sections.
- `remediation`: Optional repair plan. Required in practice whenever findings exist so `code-review-sdd` can execute without guessing.

## Summary Item

```json
{
  "label": "📦 变更规模",
  "risk": false,
  "content_html": "<ul><li>42 files changed</li></ul>"
}
```

`content_html` is inserted as trusted generated HTML. Keep it short.

## Section Block Types

### paragraph

```json
{"type": "paragraph", "text": "说明文字"}
```

### list

```json
{"type": "list", "ordered": true, "items": ["主线一", "主线二"]}
```

### table

```json
{
  "type": "table",
  "headers": ["文件/模块", "综合风险", "证据"],
  "rows": [["src/a.ts", "⚠️ 需关注", "缺少测试"]]
}
```

### mermaid

```json
{"type": "mermaid", "source": "flowchart TD\nA[变更] --> B[风险]"}
```

The self-contained HTML renderer converts compact acyclic
`flowchart TD|TB|LR|RL` diagrams with `-->`, `==>`, or `-.->` edges into inline
SVG. It keeps the escaped Mermaid source in a disclosure below the diagram.
Unsupported syntax falls back to that visible source; it never triggers a
network fetch.

Within the supported flowchart subset, a later bare node reference such as
`A --> C` must not overwrite an earlier explicit label such as `A[业务入口]`.
When the first occurrence is bare, a later explicit declaration may complete
its label. This keeps branch and merge diagrams readable regardless of edge
order.

### findings

```json
{
  "type": "findings",
  "items": [
    {
      "id": "F-001",
      "priority": "P1",
      "title": "问题标题",
      "file": "src/a.ts",
      "line": 12,
      "risk": "high",
      "detail": "影响说明",
      "recommendation": "修复建议"
    }
  ]
}
```

Priorities: `P0`, `P1`, `P2`, `P3`.

Risks: `high`, `medium`, `low`.

Give every actionable finding a stable `id`. Residual risks and non-blocking suggestions should not appear here.

## Remediation / Code Review SDD

When findings exist, include `remediation` so the renderer can emit `.code-review-sdd/code-review-sdd.md`.

```json
{
  "remediation": {
    "strategy": ["先修阻塞合入的问题", "补齐回归后再复核"],
    "tasks": [
      {
        "id": "CR-001",
        "finding_ids": ["F-001"],
        "priority": "P1",
        "title": "修复任务标题",
        "objective": "可验证目标",
        "rationale": "审查证据与影响",
        "status": "PENDING",
        "dependencies": [],
        "parallel_safe": false,
        "human_confirmation_required": false,
        "scope": {
          "allowed_paths": ["src/a.ts", "src/a.test.ts"],
          "excluded_paths": []
        },
        "source_evidence": ["src/a.ts:12"],
        "steps": ["复现", "修复", "补测试", "验证"],
        "acceptance_criteria": ["原问题不再复现", "相关测试通过"],
        "tests": ["pnpm --filter hr-frontend test -- src/a.test.ts"],
        "rollback": "通过 git 恢复本任务引入的变更",
        "deliverables": ["代码修复", "回归测试", "验证证据"]
      }
    ]
  }
}
```

Rules:

- Map every open finding to at least one task.
- Prefer one cohesive outcome per task; combine findings only when they share root cause and verification.
- Keep `allowed_paths` narrow.
- Set `human_confirmation_required: true` for public API, schema/migration, shared auth, lockfile/global config, or destructive changes.
- If `remediation.tasks` is omitted, the renderer auto-derives one task per findings item. Prefer explicit tasks for real reviews.

The generated SDD Markdown embeds a fenced `code-review-sdd-manifest` JSON block for `code-review-sdd` parsing.

When a review has zero actionable findings, the renderer atomically replaces
the canonical `.code-review-sdd/code-review-sdd.md` with a `no_action: true`
manifest containing empty findings and tasks. This marker invalidates older
timestamped plans. `--no-sdd` explicitly suppresses both actionable plans and
the no-action canonical marker.

## Commit Review Notes

When reviewing one commit, fill:

```json
{
  "scope": "commit 05547a8 (05547a8^..05547a8)",
  "scope_type": "commit",
  "reviewed_commit": "05547a8"
}
```

The report should focus on changes introduced by that commit only. Do not include unrelated working-tree changes unless the user explicitly asks.
