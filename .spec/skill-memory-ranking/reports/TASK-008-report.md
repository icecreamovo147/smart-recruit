# TASK Report - TASK-008

## 1. TASK ID

TASK-008 — 补充单测与表驱动测试

## 2. Modified File List

- `logic-grpc-service/service/skill_memory_ranking_test.go`（追加 5 个集成 / 表驱动测试）

## 3. Change Summary by File

### `skill_memory_ranking_test.go`（追加）

- `TestPoolIsolation`：验证 Skill 池与 Memory 池的 RankingSignals 互不污染；两侧的 pool_confidence 独立推断。
- `TestVectorScoreFallbackToZero`：4 个子用例（embedding_unavailable / nan_vector / inf_vector / over_one_vector）覆盖 vector 信号的边界降级。
- `TestLexicalScoreNormalization`：2 个子用例（all_match_normalized_to_one / partial_match）验证 keyword 命中按 query token 数归一化。
- `TestMetadataScoreNormalization`：4 个子用例（application_match / job_match / hr_match / hr_alone）验证 scope 命中映射。
- `TestBoostBoundaryScenarios`：3 个子用例（zero / max_skill / max_memory）覆盖 business_boost 边界。

### 累计测试覆盖（来自 TASK-001..008）

| 测试函数 | 来源 | 覆盖点 |
| --- | --- | --- |
| `TestComputeRelevanceScore` | TASK-001 | 10 个 case：纯 vector / 纯 lexical / 纯 metadata / 混合 / 全 0 / 越界 / NaN |
| `TestComputeBusinessBoost` | TASK-001 | 10 个 case：Skill priority 0/50/100/1000/-10/NaN + Memory gated/active/out-of-range |
| `TestComputeFinalRankScore` | TASK-001 | 4 个 case：零相关、满相关、混合、NaN boost |
| `TestMemoryImportanceGatedByRelevance` | TASK-001 | 4 个 case：低于阈值 / 等于阈值 / 高于阈值 / 越界 |
| `TestComputePoolConfidence` | TASK-001 + TASK-004 | 9 个 case：空 / 单元素低于 / 单元素高于 / 高 / 中 / 低 / 强制 low / NaN / 乱序 + TASK-004 修复 all_zero → none |
| `TestScoreSkillRankingSignalsAllModes` | TASK-002 | 6 个 case：vector_only / lexical_only / hybrid / degraded / no_match / nan |
| `TestRankSkillCandidatesAllModes` | TASK-002 | 5 个 case：embedding_unavailable / embedding_available / mixed_signals / zero_score / sort |
| `TestSkillPoolFallbackToLexicalMetadata` | TASK-002 | 降级端到端 |
| `TestScoreMemoryRankingSignalsAllModes` | TASK-003 | 4 个 case：vector_high / lexical_only / hybrid / nan |
| `TestRankMemoryCandidatesAllModes` | TASK-003 | 4 个 case：embedding_unavailable / application_first / zero_score / sort |
| `TestMemoryImportanceGatedByRelevance_RankMemoryCandidates` | TASK-003 | 2 个 case：low_gates / high_uses |
| `TestMemoryPoolFallbackToLexicalMetadata` | TASK-003 | 降级端到端 |
| `TestSkillPoolConfidenceFromRankings` | TASK-004 | 3 个 case：high_gap / low_gap / none_for_empty |
| `TestMemoryPoolConfidenceFromRankings` | TASK-004 | 2 个 case：medium_gap / all_zero_none |
| `TestPoolIsolation` | **TASK-008** | Skill / Memory 池互不污染 |
| `TestVectorScoreFallbackToZero` | **TASK-008** | 4 个子用例 |
| `TestLexicalScoreNormalization` | **TASK-008** | 2 个子用例 |
| `TestMetadataScoreNormalization` | **TASK-008** | 4 个子用例 |
| `TestBoostBoundaryScenarios` | **TASK-008** | 3 个子用例 |
| `TestAgentSkillServiceDebugSemanticRetrievalPopulatesScoreBreakdown` | TASK-006 | proto breakdown + pool_confidence 端到端 |
| `TestAgentSkillServiceDebugSemanticRetrievalFallbackPopulatesBreakdown` | TASK-006 | 降级路径 |

## 4. Scope Check Result

- 实际修改文件：`logic-grpc-service/service/skill_memory_ranking_test.go`，在 `task-scope.json` 的 `allowedFiles` 内。
- `bash .spec/skill-memory-ranking/scripts/agent-check.sh` → 全部通过（gofmt / go build / go test / web-gin build / hr-frontend typecheck）。

## 5. SPEC Comparison Result

| SPEC 引用 | 实现情况 |
| --- | --- |
| §5 FR-010 测试覆盖 | ✅ 全部 6 个 case 类别覆盖（纯 vector / 纯 lexical / 纯 metadata / 混合 / 全 0 / 极端值）+ Memory gating + Skill/Memory 池互不污染 + 降级路径 |
| §11 AC-001 | ✅ 6 种典型组合（实际 30+ case） |
| §11 AC-009 | ✅ `go test ./...` 全部通过 |
| §11 AC-010 | ✅ `pnpm --filter hr-frontend typecheck` 通过 |

## 6. SDD Comparison Result

- SDD §11 测试策略：4 类测试（单元 / 集成 / 表驱动 / 兼容性）全部覆盖。
- SDD §8 兼容性策略：现有测试不删除；不弱化已有断言；`Score` 强断言已迁移到 `FinalRankScore`（TASK-002 调整时已确认）。

## 7. Acceptance Comparison Result

| AC | 结果 |
| --- | --- |
| AC-001 测试覆盖 SPEC §5 FR-010 全部场景 | ✅ |
| AC-002 `go test ./...` 通过 | ✅ |
| AC-003 `pnpm --filter hr-frontend typecheck` 通过 | ✅ |
| AC-004 不删除已有测试；不弱化已有断言 | ✅ |
| AC-005 测试不依赖真实 embedding provider（使用 `fakeEmbeddingProvider` / nil provider） | ✅ |
| AC-006 覆盖纯 vector / 纯 lexical / 纯 metadata / 混合 / 全 0 / 极端值；Memory gating；top1/top2 gap 高/中/低/无；Skill/Memory 池互不污染；Embedding 降级 | ✅ |

## 8. Test Commands and Results

| 命令 | 结果 |
| --- | --- |
| `go test -v -run 'TestPoolIsolation\|TestVectorScoreFallbackToZero\|TestLexicalScoreNormalization\|TestMetadataScoreNormalization\|TestBoostBoundaryScenarios' ./service/...` | 全部 `PASS` |
| `go test ./...` (logic-grpc-service) | 全部 `ok`（11 个包全部通过） |
| `bash .spec/skill-memory-ranking/scripts/agent-check.sh` | 全部通过（gofmt / go build / go test / web-gin build / hr-frontend typecheck） |

## 9. Risks

- 无新风险。
- 测试文件累计 ~1000 行，覆盖率已与 SDD §11 测试策略要求对齐。

## 10. Follow-up Items

- 后续 PR：可考虑为 `poolConfidence` 引入 benchmark，验证 1000+ 候选下的性能。
- 后续 PR：可补充 fuzz 测试覆盖 `ComputePoolConfidence` / `clamp01` 等纯函数。

## 11. Whether the Next TASK Can Start

✅ 所有 8 个 TASK 全部完成。可以生成 pipeline-summary.md 收尾。
