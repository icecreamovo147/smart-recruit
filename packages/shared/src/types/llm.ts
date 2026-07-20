// ── LLM Provider ──────────────────────────────────────────────────────────

export interface LlmProvider {
  id: number
  name: string
  base_url: string
  api_key_masked: string   // masked key (first 4 + last 4 chars)
  provider_type: string     // e.g. "openai", "anthropic", "azure", "ollama"
  protocol_type: string
  auth_type: string
  api_version: string
  discovery_url: string
  extra_headers_json: string
  is_enabled: boolean
  created_at: string
  updated_at: string
}

export interface CreateProviderPayload {
  name: string
  base_url: string
  api_key: string
  provider_type: string
  protocol_type?: string
  auth_type?: string
  api_version?: string
  discovery_url?: string
  extra_headers_json?: string
}

export interface UpdateProviderPayload {
  name?: string
  base_url?: string
  api_key?: string
  provider_type?: string
  protocol_type?: string
  auth_type?: string
  api_version?: string
  api_version_set?: boolean
  discovery_url?: string
  discovery_url_set?: boolean
  extra_headers_json?: string
  extra_headers_set?: boolean
  is_enabled?: boolean
  is_enabled_set?: boolean
}

// ── LLM Model ─────────────────────────────────────────────────────────────

export interface LlmModel {
  id: number
  provider_id: number
  model_name: string
  catalog_model_name?: string
  display_name: string
  temperature: number
  top_p: number
  max_tokens: number
  context_window_tokens: number
  provider_max_input_tokens?: number
  provider_max_output_tokens?: number
  capabilities_json?: string
  metadata_source?: string
  metadata_sources_json?: string
  metadata_synced_at?: string
  temperature_enabled?: boolean
  top_p_enabled?: boolean
  max_concurrency: number
  timeout_seconds: number
  is_enabled: boolean
  is_default: boolean
  created_at: string
  updated_at: string
  provider_name: string   // joined field from provider table
}

export interface CreateModelPayload {
  provider_id: number
  model_name: string
  catalog_model_name?: string
  display_name: string
  temperature: number
  temperature_set?: boolean
  top_p: number
  top_p_set?: boolean
  max_tokens: number
  context_window_tokens?: number
  provider_max_input_tokens?: number
  provider_max_output_tokens?: number
  capabilities_json?: string
  metadata_source?: string
  metadata_sources_json?: string
  metadata_synced_at?: string
  temperature_enabled?: boolean
  temperature_enabled_set?: boolean
  top_p_enabled?: boolean
  top_p_enabled_set?: boolean
  max_concurrency?: number
  timeout_seconds?: number
  is_default?: boolean
}

export interface UpdateModelPayload {
  model_name?: string
  catalog_model_name?: string
  catalog_model_name_set?: boolean
  display_name?: string
  temperature?: number
  temperature_set?: boolean
  top_p?: number
  top_p_set?: boolean
  max_tokens?: number
  max_tokens_set?: boolean
  context_window_tokens?: number
  context_window_tokens_set?: boolean
  provider_max_input_tokens?: number
  provider_max_input_tokens_set?: boolean
  provider_max_output_tokens?: number
  provider_max_output_tokens_set?: boolean
  capabilities_json?: string
  capabilities_json_set?: boolean
  metadata_source?: string
  metadata_sources_json?: string
  metadata_sources_json_set?: boolean
  metadata_synced_at?: string
  temperature_enabled?: boolean
  temperature_enabled_set?: boolean
  top_p_enabled?: boolean
  top_p_enabled_set?: boolean
  max_concurrency?: number
  max_concurrency_set?: boolean
  timeout_seconds?: number
  timeout_seconds_set?: boolean
  is_enabled?: boolean
  is_enabled_set?: boolean
  is_default?: boolean
  is_default_set?: boolean
}

// ── Connection test ────────────────────────────────────────────────────────

export interface TestConnectionResult {
  success: boolean
  detail: string
}

export interface DiscoveredLlmModel {
  model_name: string
  display_name: string
  owned_by: string
  context_window_tokens: number
  context_window_tokens_known: boolean
  max_input_tokens: number
  max_input_tokens_known: boolean
  max_output_tokens: number
  max_output_tokens_known: boolean
  temperature: number
  temperature_known: boolean
  top_p: number
  top_p_known: boolean
  capabilities_json: string
  already_configured: boolean
  field_sources: ModelMetadataFieldSource[]
}

export interface ModelMetadataFieldSource {
  field_name: string
  source_type: 'provider_api' | 'provider_detail' | 'official_document' | 'platform_policy' | 'user' | string
  source_ref: string
  confidence: number
  observed_at: string
  verified: boolean
}

export interface ModelDiscoveryResult {
  list: DiscoveredLlmModel[]
  source: string
  fetched_at: string
}

export interface ModelPresetResult {
  model: DiscoveredLlmModel
  source: string
  fetched_at: string
}
