import request from './http'
import type { PaginatedList } from '@shared/types/pagination'
import type {
  CreateSkillPayload,
  CreateSkillVersionPayload,
  SkillInfo,
  SkillToolInfo,
  SkillVersionInfo,
  UpdateSkillToolPayload,
  UpdateSkillPayload,
} from '@shared/types/skill'

export const listSkills = (
  page = 1,
  pageSize = 20,
): Promise<PaginatedList<SkillInfo>> =>
  request.get('/api/v1/platform/ai/skills', { params: { page, page_size: pageSize } })

export const createSkill = (data: CreateSkillPayload): Promise<{ skill: SkillInfo }> =>
  request.post('/api/v1/platform/ai/skills', data)

export const updateSkill = (
  id: number,
  data: UpdateSkillPayload,
): Promise<{ skill: SkillInfo }> =>
  request.put(`/api/v1/platform/ai/skills/${id}`, data)

export const createSkillVersion = (
  skillId: number,
  data: CreateSkillVersionPayload,
): Promise<{ version: SkillVersionInfo; tools: SkillToolInfo[] }> =>
  request.post(`/api/v1/platform/ai/skills/${skillId}/versions`, data)

export const listSkillVersions = (
  skillId: number,
): Promise<{ list: SkillVersionInfo[] }> =>
  request.get(`/api/v1/platform/ai/skills/${skillId}/versions`)

export const activateSkillVersion = (
  skillId: number,
  versionId: number,
): Promise<{ skill: SkillInfo }> =>
  request.post(`/api/v1/platform/ai/skills/${skillId}/versions/${versionId}/activate`)

export const listSkillTools = (
  skillId: number,
  enabledOnly = false,
): Promise<{ list: SkillToolInfo[] }> =>
  request.get(`/api/v1/platform/ai/skills/${skillId}/tools`, {
    params: { enabled_only: enabledOnly ? 'true' : 'false' },
  })

export const updateSkillTool = (
  skillId: number,
  toolId: number,
  data: UpdateSkillToolPayload,
): Promise<{ tool: SkillToolInfo }> =>
  request.put(`/api/v1/platform/ai/skills/${skillId}/tools/${toolId}`, data)
