export interface EmbeddingProvider {
  id: number
  name: string
  provider_type: string
  endpoint: string
  api_key_masked: string
  extra_headers_json: string
  is_enabled: boolean
  created_at: string
  updated_at: string
}

export interface CreateEmbeddingProviderPayload {
  name: string
  provider_type: string
  endpoint: string
  api_key: string
  extra_headers_json?: string
}

export interface UpdateEmbeddingProviderPayload {
  name?: string
  provider_type?: string
  endpoint?: string
  api_key?: string
  extra_headers_json?: string
  is_enabled?: boolean
  is_enabled_set?: boolean
  extra_headers_set?: boolean
}

export interface EmbeddingModel {
  id: number
  provider_id: number
  model_name: string
  display_name: string
  embedding_dim: number
  input_token_limit: number
  batch_size: number
  timeout_seconds: number
  max_retries: number
  is_enabled: boolean
  is_default: boolean
  last_test_status: string
  last_test_error: string
  last_test_at: string
  created_at: string
  updated_at: string
  provider_name: string
}

export interface CreateEmbeddingModelPayload {
  provider_id: number
  model_name: string
  display_name?: string
  embedding_dim?: number
  input_token_limit?: number
  batch_size?: number
  timeout_seconds?: number
  max_retries?: number
  is_default?: boolean
}

export interface UpdateEmbeddingModelPayload {
  model_name?: string
  display_name?: string
  embedding_dim?: number
  embedding_dim_set?: boolean
  input_token_limit?: number
  input_token_limit_set?: boolean
  batch_size?: number
  batch_size_set?: boolean
  timeout_seconds?: number
  timeout_seconds_set?: boolean
  max_retries?: number
  max_retries_set?: boolean
  is_enabled?: boolean
  is_enabled_set?: boolean
  is_default?: boolean
  is_default_set?: boolean
}

export interface TestEmbeddingModelPayload {
  provider_id: number
  model_id: number
  test_text?: string
}

export interface TestEmbeddingModelResult {
  success: boolean
  dimension: number
  latency_ms: number
  request_id: string
  detail: string
}

export interface BackfillEmbeddingsPayload {
  object_type?: string
  object_id?: number
  limit?: number
  batch_size?: number
  force?: boolean
  dry_run?: boolean
  model_id?: number
}

export interface BackfillEmbeddingsResult {
  success_count: number
  failed_count: number
  skipped_count: number
}
