import { describe, expect, it } from 'vitest'
import {
  PROMPT_AGENT_TYPE_OPTIONS,
  findPromptAgentTypeOption,
  normalizePromptAgentType,
  promptAgentTypeKind,
  promptAgentTypeKindLabel,
  promptAgentTypeLabel,
  structuredTaskPromptAgentTypes,
} from '@shared/constants/promptAgentTypes'

describe('promptAgentTypes catalog', () => {
  it('includes conversation and structured task roles', () => {
    const values = PROMPT_AGENT_TYPE_OPTIONS.map((item) => item.value)
    expect(values).toContain('hr_recruiting_agent')
    expect(values).toContain('candidate_assistant')
    expect(values).toContain('resume_profile_extractor')
    expect(values).toContain('job_requirement_extractor')
    expect(values).toContain('candidate_match_evaluator')
    expect(structuredTaskPromptAgentTypes).toHaveLength(3)
  })

  it('maps labels and kinds for known and legacy types', () => {
    expect(promptAgentTypeLabel('candidate_match_evaluator')).toBe('人岗匹配评估')
    expect(promptAgentTypeLabel('hr_agent')).toBe('HR 招聘助手')
    expect(promptAgentTypeKind('resume_profile_extractor')).toBe('structured_task')
    expect(promptAgentTypeKindLabel('job_requirement_extractor')).toBe('系统内置任务')
    expect(promptAgentTypeKindLabel('hr_recruiting_agent')).toBe('对话助手')
  })

  it('normalizes legacy hr_agent and finds options', () => {
    expect(normalizePromptAgentType('hr_agent')).toBe('hr_recruiting_agent')
    expect(findPromptAgentTypeOption('resume_profile_extractor')?.label).toBe('简历画像抽取')
    expect(findPromptAgentTypeOption('unknown_type')).toBeUndefined()
  })
})
