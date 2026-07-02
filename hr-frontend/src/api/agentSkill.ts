import request from './request'
import type { PaginatedList } from '@/types/domain'
import type {
  AgentSkillDetailResponse,
  AgentSkillInfo,
  AgentSkillListParams,
  AgentSkillPreviewPayload,
  AgentSkillPreviewResult,
  AgentSkillResponse,
  AgentSkillVersionListResponse,
  AgentSkillVersionResponse,
  AvailableAgentSkill,
  CreateAgentSkillPayload,
  CreateAgentSkillVersionPayload,
  SemanticRetrievalDebugParams,
  SemanticRetrievalDebugResult,
  UpdateAgentSkillPayload,
  UpdateAgentSkillStatusPayload,
} from '@/types/agentSkill'

export const listAgentSkills = (
  params: AgentSkillListParams = {},
): Promise<PaginatedList<AgentSkillInfo>> =>
  request.get('/api/v1/hr/admin/agent-skills', {
    params: {
      page: params.page ?? 1,
      page_size: params.page_size ?? 20,
      ...(params.keyword ? { keyword: params.keyword } : {}),
      ...(params.enabled_only !== undefined ? { enabled_only: params.enabled_only ? 'true' : 'false' } : {}),
    },
  })

export const previewAgentSkill = (
  data: AgentSkillPreviewPayload,
): Promise<AgentSkillPreviewResult> =>
  request.post('/api/v1/hr/admin/agent-skills/preview', data)

export const getAgentSkill = (
  id: number,
): Promise<AgentSkillDetailResponse> =>
  request.get(`/api/v1/hr/admin/agent-skills/${id}`)

export const createAgentSkill = (
  data: CreateAgentSkillPayload,
): Promise<AgentSkillResponse> =>
  request.post('/api/v1/hr/admin/agent-skills', data)

export const updateAgentSkill = (
  id: number,
  data: UpdateAgentSkillPayload,
): Promise<AgentSkillResponse> =>
  request.put(`/api/v1/hr/admin/agent-skills/${id}`, data)

export const updateAgentSkillStatus = (
  id: number,
  data: UpdateAgentSkillStatusPayload,
): Promise<AgentSkillResponse> =>
  request.patch(`/api/v1/hr/admin/agent-skills/${id}/status`, data)

export const listAgentSkillVersions = (
  id: number,
): Promise<AgentSkillVersionListResponse> =>
  request.get(`/api/v1/hr/admin/agent-skills/${id}/versions`)

export const createAgentSkillVersion = (
  id: number,
  data: CreateAgentSkillVersionPayload,
): Promise<AgentSkillVersionResponse> =>
  request.post(`/api/v1/hr/admin/agent-skills/${id}/versions`, data)

export const activateAgentSkillVersion = (
  id: number,
  versionId: number,
): Promise<AgentSkillResponse> =>
  request.post(`/api/v1/hr/admin/agent-skills/${id}/versions/${versionId}/activate`)

export const listAvailableAgentSkills = (): Promise<{ list: AvailableAgentSkill[] }> =>
  request.get('/api/v1/hr/agent-skills/available')

export const debugSemanticRetrieval = (
  params: SemanticRetrievalDebugParams,
): Promise<SemanticRetrievalDebugResult> =>
  request.get('/api/v1/hr/admin/agent-skills/semantic-debug', {
    params: {
      query: params.query,
      ...(params.agent_type ? { agent_type: params.agent_type } : {}),
      ...(params.job_id ? { job_id: params.job_id } : {}),
      ...(params.application_id ? { application_id: params.application_id } : {}),
      ...(params.limit ? { limit: params.limit } : {}),
    },
  })
