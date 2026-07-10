# Semantic Retrieval Score Fixes SDD

## Design

### UI score semantics

`score` remains a compatibility field mapped to `final_rank_score`, but the HR debug view must label it as final ranking score. The raw embedding cosine score is displayed from `vector_score`.

### Skill semantic-only candidates

`RankSkillCandidates` should no longer drop a candidate solely because `scoreAgentSkillMatch` returns zero. A candidate is kept when either lexical rules match or `vector_score` reaches a conservative semantic threshold. This preserves existing lexical behavior while making the semantic debug path meaningful.

### Embedding search result quality

`rankEmbeddingRows` should deduplicate rows by `object_type/object_id/model`, retaining the highest cosine score. Candidate loading should evaluate more rows than before to reduce updated-at ordering bias before ranking.

### Skill embedding text

Introduce an extended Skill embedding text builder that includes metadata fields already used by retrieval: category, scenario, risk level, trigger keywords, semantic tags, evaluation criteria, output schema, description, and body text. Existing call sites are updated to use the extended builder.

### Proto sync

The web-gin proto source is updated to include semantic debug fields already present in generated pb files.

## Risks

- More semantic-only Skill candidates can increase recall; ranking and limit still cap the result size.
- Loading more embedding rows increases CPU work in the in-process cosine ranking path; no new dependency is introduced.
- Adding metadata to embedding text only affects newly embedded or backfilled rows; existing rows require backfill.
