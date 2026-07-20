import request from './http'
import type {
  PromptTemplate,
  CreatePromptPayload,
  UpdatePromptPayload,
  PromptVersion,
  RollbackPayload,
} from '@shared/types/prompt'
import type { PaginatedList } from '@shared/types/pagination'

// ── Template CRUD ───────────────────────────────────────────────────────────

export const listPromptTemplates = (
  page = 1,
  pageSize = 20,
  agentType?: string,
): Promise<PaginatedList<PromptTemplate>> => {
  const params: Record<string, string | number> = { page, page_size: pageSize }
  if (agentType) params.agent_type = agentType
  return request.get('/api/v1/platform/ai/prompt-templates', { params })
}

export const createPromptTemplate = (
  data: CreatePromptPayload,
): Promise<{ template: PromptTemplate }> =>
  request.post('/api/v1/platform/ai/prompt-templates', data)

export const updatePromptTemplate = (
  id: number,
  data: UpdatePromptPayload,
): Promise<{ template: PromptTemplate }> =>
  request.put(`/api/v1/platform/ai/prompt-templates/${id}`, data)

export const deletePromptTemplate = (id: number): Promise<void> =>
  request.delete(`/api/v1/platform/ai/prompt-templates/${id}`)

// ── Version Management ──────────────────────────────────────────────────────

export const getPromptVersionHistory = (
  templateId: number,
  page = 1,
  pageSize = 20,
): Promise<PaginatedList<PromptVersion>> =>
  request.get(`/api/v1/platform/ai/prompt-templates/${templateId}/versions`, {
    params: { page, page_size: pageSize },
  })

export const rollbackPromptVersion = (
  templateId: number,
  data: RollbackPayload,
): Promise<{ template: PromptTemplate }> =>
  request.post(`/api/v1/platform/ai/prompt-templates/${templateId}/rollback`, data)
