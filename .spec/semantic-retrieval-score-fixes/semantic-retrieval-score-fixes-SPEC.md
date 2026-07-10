# Semantic Retrieval Score Fixes SPEC

## Background

HR semantic retrieval debugging currently makes good semantic matches appear weak because the visible score is the hybrid final ranking score, not the raw embedding cosine score. The backend also still filters Skill candidates by lexical rule hits, truncates embedding candidates before similarity ranking, may overwrite a high score with a lower duplicate embedding for the same object, and omits several business fields from Skill embedding text.

## Goals

1. Make the HR debug UI distinguish embedding similarity from final ranking score.
2. Preserve semantic-only Skill candidates when vector similarity is strong enough.
3. Deduplicate embedding search results by object, keeping the highest score.
4. Rank embedding candidates after loading a larger candidate set, reducing updated-at truncation bias.
5. Expand Skill embedding text with category, scenario, trigger keywords, risk level, evaluation criteria, and output schema.
6. Synchronize web-gin proto source with generated debug fields.

## Non-Goals

- Do not replace the configured embedding provider or cosine similarity implementation.
- Do not introduce a vector database or new dependencies.
- Do not change existing public gRPC method signatures.
- Do not redesign the whole debug page.

## Acceptance Criteria

- HR debug cards show both `vector_score` and `final_rank_score`, and labels no longer call final rank an embedding match score.
- Skill ranking allows candidates with strong `vector_score` even when `rawRuleScore == 0`.
- Embedding search deduplicates repeated embeddings for the same object by keeping the highest score.
- Embedding candidate loading is less biased by recent rows and can evaluate a larger set for debug/topK retrieval.
- Skill embedding text includes operational metadata fields used by HR queries.
- `web-gin-service/proto/recruitment.proto` includes the same semantic debug fields as logic-grpc.
- Targeted Go tests and HR frontend typecheck pass or any failure is reported.
