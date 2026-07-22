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

### findings

```json
{
  "type": "findings",
  "items": [
    {
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
