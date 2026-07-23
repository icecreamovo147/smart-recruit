import request from './request'
import type { AvailableAgentSkill } from '@shared/types/agentSkill'

// Business users may select published Agent Skills but cannot manage their
// definitions or versions from the enterprise workspace.
export const listAvailableAgentSkills = (): Promise<{ list: AvailableAgentSkill[] }> =>
  request.get('/api/v1/hr/agent-skills/available')
