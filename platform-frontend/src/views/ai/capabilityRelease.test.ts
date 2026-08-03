import { describe, expect, it } from 'vitest'
import type { PlatformAICapabilitySnapshotV2 } from '@/api/platformAI'
import {
  buildCapabilitySnapshot,
  capabilityReleaseErrorMessage,
  createCapabilityReleaseForm,
  evaluationGateStatus,
  serializeCapabilitySnapshot,
  validateCapabilitySnapshot,
  validateSkillComposition,
  type AgentSkillReleaseOption,
} from './capabilityRelease'

const capability = {
  capability_key: 'ai.agent_run',
  audience: 'tenant_hr' as const,
}

const skillOption = (
  value: number,
  role: 'primary' | 'supporting',
  overrides: Partial<AgentSkillReleaseOption> = {},
): AgentSkillReleaseOption => ({
  value,
  label: `Skill V${value}`,
  skillId: value,
  skillEnabled: true,
  version: `${value}.0.0`,
  compiledHash: String(value).repeat(64).slice(0, 64),
  agentType: 'hr_recruiting_agent',
  scenario: 'candidate_screening',
  role,
  risk: 'medium',
  coreEstimatedTokens: 120,
  packageEstimatedTokens: 360,
  ...overrides,
})

const validForm = () => {
  const form = createCapabilityReleaseForm()
  form.allowed_llm_model_ids = [9, 3, 9]
  form.default_llm_model_id = 3
  form.allowed_embedding_model_ids = [8]
  form.default_embedding_model_id = 8
  form.agent_ids = [20]
  form.prompt_template_ids = [30]
  return form
}

describe('AI capability Package v2 release contract', () => {
  it('serializes exactly the six schema v2 top-level fields with fixed policy values', () => {
    const form = validForm()
    form.agent_skill_version_ids = [102, 101, 102]
    const snapshot = buildCapabilitySnapshot(capability, form)

    expect(Object.keys(snapshot).sort()).toEqual([
      'audience',
      'capability_key',
      'configuration_refs',
      'model_policy',
      'schema_version',
      'skill_runtime_policy',
    ])
    expect(snapshot).toMatchObject({
      schema_version: 2,
      capability_key: 'ai.agent_run',
      audience: 'tenant_hr',
      model_policy: {
        allowed_llm_model_ids: [3, 9],
        default_llm_model_id: 3,
      },
      configuration_refs: {
        agent_skill_version_ids: [101, 102],
      },
      skill_runtime_policy: {
        policy_version: 'skill-package-v2.1',
        max_skill_tokens: 3000,
        max_input_ratio: 0.15,
        max_skills: 2,
        evaluation_suite_hash: '',
        evaluation_result_hash: '',
      },
    })
    expect(JSON.parse(serializeCapabilitySnapshot(capability, form))).toEqual(snapshot)
  })

  it('enforces token and ratio boundaries while max skills remains fixed', () => {
    const form = validForm()
    const skillOptions: AgentSkillReleaseOption[] = []

    form.max_skill_tokens = 1
    form.max_input_ratio = 0.01
    expect(validateCapabilitySnapshot(buildCapabilitySnapshot(capability, form), skillOptions, false).valid).toBe(true)

    form.max_skill_tokens = 3001
    expect(validateCapabilitySnapshot(buildCapabilitySnapshot(capability, form), skillOptions, false).errors)
      .toContain('Skill Token 上限必须是 1 到 3000 的整数。')

    form.max_skill_tokens = 3000
    for (const invalidRatio of [0, Number.NaN, Number.POSITIVE_INFINITY, Number.NEGATIVE_INFINITY]) {
      form.max_input_ratio = invalidRatio
      expect(validateCapabilitySnapshot(buildCapabilitySnapshot(capability, form), skillOptions, false).errors)
        .toContain('Skill 输入占比必须大于 0 且不超过 0.15。')
    }

    form.max_input_ratio = 0.15
    expect(validateCapabilitySnapshot(buildCapabilitySnapshot(capability, form), skillOptions, false).valid).toBe(true)

    form.max_input_ratio = 0.1500001
    expect(validateCapabilitySnapshot(buildCapabilitySnapshot(capability, form), skillOptions, false).errors)
      .toContain('Skill 输入占比必须大于 0 且不超过 0.15。')

    const snapshot = buildCapabilitySnapshot(capability, validForm())
    ;(snapshot.skill_runtime_policy as { max_skills: number }).max_skills = 1
    expect(validateCapabilitySnapshot(snapshot, skillOptions, false).errors)
      .toContain('每次运行最多 Skill 数量固定为 2。')
  })

  it('prevents orphan Supporting and more than one Primary in one agent type and scenario', () => {
    const options = [
      skillOption(101, 'primary'),
      skillOption(102, 'primary'),
      skillOption(103, 'supporting'),
    ]

    expect(validateSkillComposition([103], options)).toContain(
      'Agent 类型 hr_recruiting_agent、场景 candidate_screening 的 Supporting Skill 必须与一个 Primary Skill 同时发布。',
    )
    expect(validateSkillComposition([101, 102], options)).toContain(
      'Agent 类型 hr_recruiting_agent、场景 candidate_screening 最多发布一个 Primary Skill。',
    )
    expect(validateSkillComposition([101, 103], options)).toEqual([])
  })

  it.each([
    'resume_profile_extractor',
    'job_requirement_extractor',
    'candidate_match_evaluator',
  ])('allows only one Primary across scenarios for structured agent type %s', (agentType) => {
    const options = [
      skillOption(101, 'primary', { agentType, scenario: 'screening' }),
      skillOption(102, 'primary', { agentType, scenario: 'ranking' }),
    ]

    expect(validateSkillComposition([101, 102], options)).toContain(
      `结构化 Agent 类型 ${agentType} 最多发布一个 Primary Skill。`,
    )
  })

  it('blocks disabled and missing exact Skill versions', () => {
    const disabled = skillOption(101, 'primary', { skillEnabled: false })
    expect(validateSkillComposition([101, 999], [disabled])).toEqual([
      'Skill V101 所属 Skill 已停用。',
      'Agent Skill 版本 #999 不存在或未加载。',
    ])
  })

  it('allows an unevaluated draft but disables publication until both stored hashes are valid', () => {
    const snapshot = buildCapabilitySnapshot(capability, validForm())
    const draftValidation = validateCapabilitySnapshot(snapshot, [], false)
    const publishValidation = validateCapabilitySnapshot(snapshot, [], true)

    expect(draftValidation.valid).toBe(true)
    expect(publishValidation.valid).toBe(false)
    expect(evaluationGateStatus(snapshot)).toMatchObject({ passed: false, label: '未通过' })

    snapshot.skill_runtime_policy.evaluation_suite_hash = 'a'.repeat(64)
    snapshot.skill_runtime_policy.evaluation_result_hash = 'b'.repeat(64)
    expect(validateCapabilitySnapshot(snapshot, [], true).valid).toBe(true)
    expect(evaluationGateStatus(snapshot)).toMatchObject({ passed: true, label: '通过' })
  })

  it('rejects snapshots with any seventh top-level field', () => {
    const snapshot = buildCapabilitySnapshot(capability, validForm()) as PlatformAICapabilitySnapshotV2 & { mode?: string }
    snapshot.mode = 'legacy'
    expect(validateCapabilitySnapshot(snapshot, [], false).errors).toContain(
      '发布快照必须且只能包含 schema_version、capability_key、audience、model_policy、configuration_refs、skill_runtime_policy。',
    )
  })

  it('rejects a snapshot whose immutable identity differs from the selected capability', () => {
    const snapshot = buildCapabilitySnapshot(capability, validForm())
    snapshot.capability_key = 'ai.chat'
    expect(validateCapabilitySnapshot(snapshot, [], false, capability).errors).toContain(
      '发布快照的 Capability Key 或 Audience 与当前能力不一致。',
    )
  })

  it('explains composition and deterministic evaluation backend errors', () => {
    expect(capabilityReleaseErrorMessage(new Error(
      'Agent Skill composition requires exactly one Primary',
    ))).toContain('Supporting Skill')
    expect(capabilityReleaseErrorMessage(new Error(
      'Agent Skill release evaluation did not pass',
    ))).toContain('确定性评测未通过')
  })
})
