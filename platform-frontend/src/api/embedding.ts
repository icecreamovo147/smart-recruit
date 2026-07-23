import request from './http'
import type {
  EmbeddingProvider,
  EmbeddingModel,
  CreateEmbeddingProviderPayload,
  UpdateEmbeddingProviderPayload,
  CreateEmbeddingModelPayload,
  UpdateEmbeddingModelPayload,
  TestEmbeddingModelPayload,
  TestEmbeddingModelResult,
  BackfillEmbeddingsPayload,
  BackfillEmbeddingsResult,
} from '@shared/types/embedding'
import type { PaginatedList } from '@shared/types/pagination'

export const listEmbeddingProviders = (page = 1, pageSize = 20): Promise<PaginatedList<EmbeddingProvider>> =>
  request.get('/api/v1/platform/ai/embedding-providers', { params: { page, page_size: pageSize } })

export const createEmbeddingProvider = (data: CreateEmbeddingProviderPayload): Promise<{ provider: EmbeddingProvider }> =>
  request.post('/api/v1/platform/ai/embedding-providers', data)

export const updateEmbeddingProvider = (id: number, data: UpdateEmbeddingProviderPayload): Promise<{ provider: EmbeddingProvider }> =>
  request.put(`/api/v1/platform/ai/embedding-providers/${id}`, data)

export const deleteEmbeddingProvider = (id: number): Promise<void> =>
  request.delete(`/api/v1/platform/ai/embedding-providers/${id}`)

export const listEmbeddingModels = (page = 1, pageSize = 20, providerId?: number): Promise<PaginatedList<EmbeddingModel>> => {
  const params: Record<string, number> = { page, page_size: pageSize }
  if (providerId) params.provider_id = providerId
  return request.get('/api/v1/platform/ai/embedding-models', { params })
}

export const createEmbeddingModel = (data: CreateEmbeddingModelPayload): Promise<{ model: EmbeddingModel }> =>
  request.post('/api/v1/platform/ai/embedding-models', data)

export const updateEmbeddingModel = (id: number, data: UpdateEmbeddingModelPayload): Promise<{ model: EmbeddingModel }> =>
  request.put(`/api/v1/platform/ai/embedding-models/${id}`, data)

export const setDefaultEmbeddingModel = (id: number): Promise<void> =>
  request.post(`/api/v1/platform/ai/embedding-models/${id}/set-default`)

export const testEmbeddingModel = (data: TestEmbeddingModelPayload): Promise<TestEmbeddingModelResult> =>
  request.post('/api/v1/platform/ai/embedding-models/test', data)

export const backfillEmbeddings = (data: BackfillEmbeddingsPayload): Promise<BackfillEmbeddingsResult> =>
  request.post('/api/v1/platform/ai/embedding-backfill', data)
