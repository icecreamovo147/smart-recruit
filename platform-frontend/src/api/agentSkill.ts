import request from './http'
import type { PaginatedList } from '@shared/types/pagination'
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
} from '@shared/types/agentSkill'
import { debugLog } from '@shared/utils/debugLog'

export const listAgentSkills = async (
  params: AgentSkillListParams = {},
): Promise<PaginatedList<AgentSkillInfo>> => {
  debugLog.skill.info('listAgentSkills_started', { page: params.page, page_size: params.page_size, keyword: params.keyword })
  const res: any = await request.get('/api/v1/platform/ai/agent-skills', {
    params: {
      page: params.page ?? 1,
      page_size: params.page_size ?? 20,
      ...(params.keyword ? { keyword: params.keyword } : {}),
      ...(params.enabled_only !== undefined ? { enabled_only: params.enabled_only ? 'true' : 'false' } : {}),
    },
  })
  debugLog.skill.info('listAgentSkills_succeeded', { total: res.total || 0, returned: (res.list || []).length })
  return res
}

export const previewAgentSkill = async (
  data: AgentSkillPreviewPayload,
): Promise<AgentSkillPreviewResult> => {
  debugLog.skill.info('previewAgentSkill_started', { skill_name: data.name, version: data.version })
  const res: any = await request.post('/api/v1/platform/ai/agent-skills/preview', data)
  debugLog.skill.info('previewAgentSkill_finished', { has_skill_md: !!res.skill_md, valid: res.validation?.valid })
  return res
}

export const getAgentSkill = async (
  id: number,
): Promise<AgentSkillDetailResponse> => {
  debugLog.skill.info('getAgentSkill_started', { skill_id: id })
  const res: any = await request.get(`/api/v1/platform/ai/agent-skills/${id}`)
  debugLog.skill.info('getAgentSkill_succeeded', { skill_id: id, has_detail: !!res })
  return res
}

export const createAgentSkill = async (
  data: CreateAgentSkillPayload,
): Promise<AgentSkillResponse> => {
  debugLog.skill.info('createAgentSkill_started', { name: data.name, display_name: data.display_name, version: data.version })
  const res: any = await request.post('/api/v1/platform/ai/agent-skills', data)
  debugLog.skill.info('createAgentSkill_succeeded', { name: data.name, skill_id: res?.skill?.id ?? res?.id })
  return res
}

export const updateAgentSkill = async (
  id: number,
  data: UpdateAgentSkillPayload,
): Promise<AgentSkillResponse> => {
  debugLog.skill.info('updateAgentSkill_started', { skill_id: id })
  const res: any = await request.put(`/api/v1/platform/ai/agent-skills/${id}`, data)
  debugLog.skill.info('updateAgentSkill_succeeded', { skill_id: id })
  return res
}

export const updateAgentSkillStatus = async (
  id: number,
  data: UpdateAgentSkillStatusPayload,
): Promise<AgentSkillResponse> => {
  debugLog.skill.info('updateAgentSkillStatus_started', { skill_id: id, is_enabled: data.is_enabled })
  const res: any = await request.patch(`/api/v1/platform/ai/agent-skills/${id}/status`, data)
  debugLog.skill.info('updateAgentSkillStatus_succeeded', { skill_id: id, is_enabled: data.is_enabled })
  return res
}

export const regenerateAgentSkillEmbedding = async (
  id: number,
): Promise<{ success_count: number; failed_count: number; skipped_count: number }> => {
  debugLog.skill.info('regenerateAgentSkillEmbedding_started', { skill_id: id })
  const res: any = await request.post(`/api/v1/platform/ai/agent-skills/${id}/embedding/regenerate`)
  debugLog.skill.info('regenerateAgentSkillEmbedding_succeeded', {
    skill_id: id,
    success_count: res?.success_count || 0,
    failed_count: res?.failed_count || 0,
    skipped_count: res?.skipped_count || 0,
  })
  return res
}

export const listAgentSkillVersions = async (
  id: number,
): Promise<AgentSkillVersionListResponse> => {
  debugLog.skill.info('listAgentSkillVersions_started', { skill_id: id })
  const res: any = await request.get(`/api/v1/platform/ai/agent-skills/${id}/versions`)
  debugLog.skill.info('listAgentSkillVersions_succeeded', { skill_id: id, version_count: (res.list || []).length })
  return res
}

export const createAgentSkillVersion = async (
  id: number,
  data: CreateAgentSkillVersionPayload,
): Promise<AgentSkillVersionResponse> => {
  debugLog.skill.info('createAgentSkillVersion_started', { skill_id: id, version: data.version })
  const res: any = await request.post(`/api/v1/platform/ai/agent-skills/${id}/versions`, data)
  debugLog.skill.info('createAgentSkillVersion_succeeded', { skill_id: id, version_id: res?.version?.id ?? res?.id })
  return res
}

export const activateAgentSkillVersion = async (
  id: number,
  versionId: number,
): Promise<AgentSkillResponse> => {
  debugLog.skill.info('activateAgentSkillVersion_started', { skill_id: id, version_id: versionId })
  const res: any = await request.post(`/api/v1/platform/ai/agent-skills/${id}/versions/${versionId}/activate`)
  debugLog.skill.info('activateAgentSkillVersion_succeeded', { skill_id: id, version_id: versionId })
  return res
}

export const listAvailableAgentSkills = async (): Promise<{ list: AvailableAgentSkill[] }> => {
  debugLog.skill.info('listAvailableAgentSkills_started', {})
  const res: any = await request.get('/api/v1/hr/agent-skills/available')
  debugLog.skill.info('listAvailableAgentSkills_succeeded', { count: (res.list || []).length })
  return res
}

export const debugSemanticRetrieval = async (
  params: SemanticRetrievalDebugParams,
): Promise<SemanticRetrievalDebugResult> => {
  debugLog.skill.info('debugSemanticRetrieval_started', { query_length: params.query.length, agent_type: params.agent_type })
  const res: any = await request.get('/api/v1/platform/ai/agent-skills/semantic-debug', {
    params: {
      query: params.query,
      ...(params.agent_type ? { agent_type: params.agent_type } : {}),
      ...(params.job_id ? { job_id: params.job_id } : {}),
      ...(params.application_id ? { application_id: params.application_id } : {}),
      ...(params.limit ? { limit: params.limit } : {}),
      ...(params.owner_role ? { owner_role: params.owner_role } : {}),
      ...(params.owner_id ? { owner_id: params.owner_id } : {}),
    },
  })
  debugLog.skill.info('debugSemanticRetrieval_succeeded', { skill_count: (res.skills || []).length, memory_count: (res.memories || []).length })
  return res
}
