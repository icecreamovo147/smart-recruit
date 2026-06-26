// ── LLM Provider ──────────────────────────────────────────────────────────

export interface LlmProvider {
  id: number
  name: string
  base_url: string
  api_key_masked: string   // masked key (first 4 + last 4 chars)
  provider_type: string     // e.g. "openai", "anthropic", "azure", "ollama"
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
  extra_headers_json?: string
}

export interface UpdateProviderPayload {
  name?: string
  base_url?: string
  api_key?: string
  provider_type?: string
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
  display_name: string
  temperature: number
  top_p: number
  max_tokens: number
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
  display_name: string
  temperature: number
  top_p: number
  max_tokens: number
  max_concurrency?: number
  timeout_seconds?: number
  is_default?: boolean
}

export interface UpdateModelPayload {
  model_name?: string
  display_name?: string
  temperature?: number
  temperature_set?: boolean
  top_p?: number
  top_p_set?: boolean
  max_tokens?: number
  max_tokens_set?: boolean
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
