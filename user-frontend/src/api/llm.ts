import request from './request'
import type { LlmModel, PaginatedList } from '@/types/llm'

export const listAvailableModels = (page = 1, pageSize = 200): Promise<PaginatedList<LlmModel>> =>
  request.get('/api/v1/candidate/ai/models', { params: { page, page_size: pageSize } })
