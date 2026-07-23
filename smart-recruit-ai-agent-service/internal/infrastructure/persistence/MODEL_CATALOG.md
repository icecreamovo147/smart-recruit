# Bundled LLM model catalog

`model_catalog_data.json` contains reviewed metadata that provider list APIs do
not expose. It is reference data, not a runtime configuration: users can edit
all values before saving an `llm_models` row.

Maintenance rules:

1. Use only official provider documentation or a provider detail API.
2. Update `revision` whenever any entry changes.
3. Set `verified_at` to the review date and use a finite `expires_at` so stale
   facts become unknown instead of silently remaining authoritative.
4. Use `null`/omit a field when the provider does not state it. Never encode
   unknown limits as zero.
5. Put field-specific evidence URLs in `field_sources` when they differ from
   the entry-level `source_url`.
6. Keep `(provider_family, model_name)` unique. For recognized providers the
   family is derived from the API hostname; otherwise it falls back to the
   canonical `provider_type`.

The AI Agent service validates and imports the file on startup. Import is an
idempotent upsert by `(provider_family, model_name)` and never overwrites rows
whose `managed_by` value is not `bundled`. Entries removed from the file are
marked inactive rather than deleted.

The bundled catalog currently covers the public model identifiers of DeepSeek,
OpenAI, Anthropic, Google Gemini, xAI, Mistral, Alibaba Cloud Model Studio
(Qwen), Zhipu GLM, MiniMax, Xiaomi MiMo, and Moonshot Kimi. For each provider,
the catalog includes both current models and older model IDs that remain
callable. Models whose official shutdown date has passed are not kept active.
Azure OpenAI deployments, Ollama installations, and provider platforms that
expose tenant-specific deployment or endpoint IDs are intentionally not
bundled; their model names must come from live discovery for the configured
provider instance.
