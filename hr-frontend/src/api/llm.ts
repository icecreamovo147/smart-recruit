import request from './request'
import type {
  LlmProvider,
  LlmModel,
  CreateProviderPayload,
  UpdateProviderPayload,
  CreateModelPayload,
  UpdateModelPayload,
  TestConnectionResult,
  ModelDiscoveryResult,
  ModelPresetResult,
} from '@/types/llm'
import type { PaginatedList } from '@/types/domain'

// ── Provider CRUD ─────────────────────────────────────────────────────────

export const listProviders = (page = 1, pageSize = 20): Promise<PaginatedList<LlmProvider>> =>
  request.get('/api/v1/hr/admin/llm-providers', { params: { page, page_size: pageSize } })

export const createProvider = (data: CreateProviderPayload): Promise<{ provider: LlmProvider }> =>
  request.post('/api/v1/hr/admin/llm-providers', data)

export const updateProvider = (id: number, data: UpdateProviderPayload): Promise<{ provider: LlmProvider }> =>
  request.put(`/api/v1/hr/admin/llm-providers/${id}`, data)

export const deleteProvider = (id: number): Promise<void> =>
  request.delete(`/api/v1/hr/admin/llm-providers/${id}`)

/** @deprecated Prefer testModelConnection — provider-level test is retained for compatibility. */
export const testProviderConnection = (id: number): Promise<TestConnectionResult> =>
  request.post(`/api/v1/hr/admin/llm-providers/${id}/test`)

export const discoverProviderModels = (id: number, refresh = false): Promise<ModelDiscoveryResult> =>
  request.post(`/api/v1/hr/admin/llm-providers/${id}/models/discover`, { refresh })

export const getProviderModelPreset = (id: number, modelName: string): Promise<ModelPresetResult> =>
  request.post(`/api/v1/hr/admin/llm-providers/${id}/models/preset`, { model_name: modelName })

// ── Model CRUD ────────────────────────────────────────────────────────────

export const listModels = (page = 1, pageSize = 20, providerId?: number): Promise<PaginatedList<LlmModel>> => {
  const params: Record<string, number> = { page, page_size: pageSize }
  if (providerId) params.provider_id = providerId
  return request.get('/api/v1/hr/admin/llm-models', { params })
}

export const listAvailableModels = (page = 1, pageSize = 200): Promise<PaginatedList<LlmModel>> =>
  request.get('/api/v1/hr/ai/models', { params: { page, page_size: pageSize } })

export const createModel = (data: CreateModelPayload): Promise<{ model: LlmModel }> =>
  request.post('/api/v1/hr/admin/llm-models', data)

export const updateModel = (id: number, data: UpdateModelPayload): Promise<{ model: LlmModel }> =>
  request.put(`/api/v1/hr/admin/llm-models/${id}`, data)

export const deleteModel = (id: number): Promise<void> =>
  request.delete(`/api/v1/hr/admin/llm-models/${id}`)

export const testModelConnection = (id: number): Promise<TestConnectionResult> =>
  request.post(`/api/v1/hr/admin/llm-models/${id}/test`)
