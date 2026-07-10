# Agent Rules

- Keep the fix scoped to semantic retrieval scoring and debug visibility.
- Do not introduce new dependencies.
- Do not modify generated protobuf files in this task.
- Preserve backward compatibility for `score` while making the UI label explicit.
- Add focused regression tests for each backend scoring behavior change.
