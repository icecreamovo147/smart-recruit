import { describe, expect, it } from 'vitest'
import { isCompatibleAgentPrompt } from './AgentManageView.vue'
import type { PromptTemplate } from '@/types/prompt'

const prompt = (overrides: Partial<PromptTemplate> = {}): PromptTemplate => ({
  id: 1,
  name: 'prompt',
  content: 'system',
  variables_json: '[]',
  version: 1,
  is_active: true,
  agent_type: 'hr_recruiting_agent',
  prompt_role: 'system',
  created_by: 1,
  updated_by: 1,
  created_at: '',
  updated_at: '',
  ...overrides,
})
describe('AgentManage Prompt selector compatibility', () => {
  it('includes exact and legacy active HR system Prompts', () => {
    expect(isCompatibleAgentPrompt(prompt(), 'hr_recruiting_agent')).toBe(true)
    expect(isCompatibleAgentPrompt(prompt({ agent_type: 'hr_agent' }), 'hr_recruiting_agent')).toBe(true)
  })

  it('excludes inactive, non-system, and incompatible Prompts', () => {
    expect(isCompatibleAgentPrompt(prompt({ is_active: false }), 'hr_recruiting_agent')).toBe(false)
    expect(isCompatibleAgentPrompt(prompt({ prompt_role: 'user' }), 'hr_recruiting_agent')).toBe(false)
    expect(isCompatibleAgentPrompt(prompt({ agent_type: 'candidate_assistant' }), 'hr_recruiting_agent')).toBe(false)
  })
})
