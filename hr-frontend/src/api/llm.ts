import request from './request'
import type { LlmModel } from '@shared/types/llm'
import type { PaginatedList } from '@/types/domain'

// Runtime selection remains in the enterprise workspace; the returned models
// are already filtered by the purchased platform capability release.
export const listAvailableModels = (page = 1, pageSize = 200): Promise<PaginatedList<LlmModel>> =>
  request.get('/api/v1/hr/ai/models', { params: { page, page_size: pageSize } })
