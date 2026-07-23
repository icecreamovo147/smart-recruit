---
name: code-review-leader
description: Review current Git branch, working tree, staged/unstaged/untracked changes, or one specific commit on the current branch like a senior code review lead, then generate paired Chinese review reports in both self-contained HTML and Markdown. Use when the user asks to inspect current branch changes, review a specific commit, compare against base branches or commits, produce Code Review Leader reports, create HTML/MD review reports similar to a provided sample, summarize architecture/risk/test impacts, or generate actionable code-review findings.
---

# Code Review Leader

## Core Workflow

Use this project-level skill to review code changes and write paired report artifacts: one visual HTML report and one Markdown report with the same review conclusions.

1. Establish the review scope.
   - Prefer the user's explicit scope.
   - For a specific commit on the current branch, use `git show --stat <commit>`, `git show --name-only <commit>`, and `git show --find-renames --find-copies --stat --patch <commit>` as the primary evidence. Treat the reviewed range as `<commit>^..<commit>` unless the user specifies a different parent/range.
   - If not specified, inspect `git status --short`, `git branch --show-current`, `git merge-base HEAD origin/main`, `git diff`, `git diff --cached`, and untracked files.
   - If the remote/base is unavailable, fall back to `HEAD` versus working tree and state that scope.
   - State clearly whether the report covers a branch/worktree range or one commit.

2. Collect evidence before judging.
   - Read changed files and relevant neighbors.
   - Use `rg` for source search.
   - Categorize files as production, test, docs/config, generated/spec artifacts, and infrastructure.
   - Identify commit range, changed file count, insertions/deletions, key directories, public API/schema/config changes, and test coverage changes.

3. Review like a lead.
   - Prioritize correctness, security, data integrity, API compatibility, performance, maintainability, and missing tests.
   - Include only actionable findings the author would likely fix.
   - Tie each issue to file/line evidence.
   - Separate confirmed issues from residual risks and follow-up suggestions.
   - Do not invent evidence. If a check was not run, say so.

4. Run focused validation when practical.
   - Use repository guidance first (`AGENTS.md`, package scripts, task contracts).
   - Prefer targeted tests/typechecks related to changed areas.
   - Record commands and outcomes in the report.

5. Generate the report JSON, then render HTML and Markdown together. By default the script writes both files to `.code-review-sdd/`:
   `node .agents/skills/code-review-leader/scripts/render_review_report.mjs --input <report.json>`

6. Return both report paths and a concise summary. If the report contains findings, list the highest priority items first.

## Report Shape

The HTML should be Chinese and visually structured like the sample report. The Markdown report should contain the same core information in a review-friendly text format.

- Title: `Code Review Leader — 变更可视化报告`
- Top metadata: generated time, repository name, branch, base/scope
- Summary bar:
  - 审查范围
  - 变更规模
  - 核心改动主线
  - 整体评估
- Collapsible sections:
  - `🔗 D1: 变更逻辑链`
  - `🏗️ D2: 架构影响分析`
  - `🔄 D3: 关键路径时序`
  - `📐 D4: 模型与契约变化`
  - `🎯 D5: 风险热力矩阵`
  - `🧪 验证与测试`
  - `📚 知识库/规则对齐`
  - `🎬 AI 建议`

Include Mermaid diagrams only when they clarify relationships or flow. Keep diagrams compact enough to render.

## Report JSON

Create a temporary JSON file with this shape:

```json
{
  "title": "Code Review Leader — 变更可视化报告",
  "repo": "smart-recruit",
  "branch": "feature/example",
  "generated_at": "2026-07-11 14:30 CST",
  "scope": "working tree vs HEAD / commit abc1234",
  "scope_type": "branch | working-tree | commit | range",
  "reviewed_commit": "abc1234",
  "verdict": "可合入 / 需修复后合入 / 不建议合入",
  "risk_level": "低风险 / 需重点关注 / 高风险",
  "summary": [
    {"label": "🔍 审查范围", "content_html": "<p>...</p>"},
    {"label": "📦 变更规模", "content_html": "<ul><li>...</li></ul>"},
    {"label": "⚡ 整体评估", "risk": true, "content_html": "<ul><li>...</li></ul>"}
  ],
  "sections": [
    {
      "title": "🎯 D5: 风险热力矩阵",
      "blocks": [
        {"type": "paragraph", "text": "本节说明..."},
        {"type": "table", "headers": ["文件/模块", "风险", "证据"], "rows": [["src/a.ts", "⚠️", "line 12"]]},
        {"type": "findings", "items": [
          {
            "priority": "P1",
            "title": "问题标题",
            "file": "src/a.ts",
            "line": 12,
            "risk": "high",
            "detail": "问题说明",
            "recommendation": "修复建议"
          }
        ]}
      ]
    }
  ]
}
```

`content_html` is allowed for rich summary text. In section blocks, prefer structured block types over raw HTML.

## Rendering Script

Use `scripts/render_review_report.mjs`.

- `--input`: report JSON path.
- `--output`: optional final HTML path. Defaults to `.code-review-sdd/code-review-leader-YYYYMMDD-HHMMSS.html`.
- `--markdown-output`: optional final Markdown report path. Defaults to the same basename as the HTML report with `.md`.
- `--sample`: write sample HTML/Markdown reports without an input file, useful for smoke tests.

Supported section block types:

- `paragraph`: `{ "type": "paragraph", "text": "..." }`
- `html`: `{ "type": "html", "html": "<p>trusted generated HTML</p>" }`
- `list`: `{ "type": "list", "ordered": false, "items": ["..."] }`
- `table`: `{ "type": "table", "headers": ["..."], "rows": [["..."]] }`
- `mermaid`: `{ "type": "mermaid", "source": "flowchart TD\nA-->B" }`
- `findings`: `{ "type": "findings", "items": [...] }`

Escape user/source text unless intentionally emitting generated HTML.

## Output Location

Default to a timestamped file under the repository:

`.code-review-sdd/code-review-leader-YYYYMMDD-HHMMSS.html`
`.code-review-sdd/code-review-leader-YYYYMMDD-HHMMSS.md`

Create the directory if missing. `.code-review-sdd/` must remain ignored by git and must not be committed. Do not overwrite an existing report unless the user asks.
