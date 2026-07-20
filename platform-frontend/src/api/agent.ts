import request from './http'
import type {
  AgentConfigInfo,
  CapabilityInfo,
  CreateAgentPayload,
  UpdateAgentPayload,
} from '@shared/types/agent'
import type { PaginatedList } from '@shared/types/pagination'

// ── Agent Config CRUD ──────────────────────────────────────────────────

export const listAgentConfigs = (
  page = 1,
  pageSize = 20,
  agentType?: string,
): Promise<PaginatedList<AgentConfigInfo>> => {
  const params: Record<string, string | number> = { page, page_size: pageSize }
  if (agentType) params.agent_type = agentType
  return request.get('/api/v1/platform/ai/agent-configs', { params })
}

export const listAgentCapabilities = (
  agentType?: string,
): Promise<{ list: CapabilityInfo[] }> => {
  const params: Record<string, string> = {}
  if (agentType) params.agent_type = agentType
  return request.get('/api/v1/platform/ai/agent-configs/capabilities', { params })
}

export const createAgentConfig = (
  data: CreateAgentPayload,
): Promise<{ agent: AgentConfigInfo }> =>
  request.post('/api/v1/platform/ai/agent-configs', data)

export const updateAgentConfig = (
  id: number,
  data: UpdateAgentPayload,
): Promise<{ agent: AgentConfigInfo }> =>
  request.put(`/api/v1/platform/ai/agent-configs/${id}`, data)

export const deleteAgentConfig = (id: number): Promise<void> =>
  request.delete(`/api/v1/platform/ai/agent-configs/${id}`)
