import request from './request'
import type { PaginatedList } from '@/types/domain'
import type {
  AgentSkillInfo,
  AgentSkillListParams,
  AgentSkillPreviewPayload,
  AgentSkillPreviewResult,
  AvailableAgentSkill,
  CreateAgentSkillPayload,
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

export const createAgentSkill = (
  data: CreateAgentSkillPayload,
): Promise<{ skill: AgentSkillInfo }> =>
  request.post('/api/v1/hr/admin/agent-skills', data)

export const updateAgentSkillStatus = (
  id: number,
  data: UpdateAgentSkillStatusPayload,
): Promise<{ skill: AgentSkillInfo }> =>
  request.patch(`/api/v1/hr/admin/agent-skills/${id}/status`, data)

export const listAvailableAgentSkills = (): Promise<{ list: AvailableAgentSkill[] }> =>
  request.get('/api/v1/hr/agent-skills/available')
