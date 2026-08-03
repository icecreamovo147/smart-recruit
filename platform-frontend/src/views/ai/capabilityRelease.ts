import type {
  AgentSkillInfo,
  AgentSkillRiskLevel,
  AgentSkillCompositionRole,
  AgentSkillVersionInfo,
} from '@shared/types/agentSkill'
import {
  PLATFORM_AI_CAPABILITY_SCHEMA_VERSION,
  PLATFORM_AI_MAX_INPUT_RATIO,
  PLATFORM_AI_MAX_SKILLS,
  PLATFORM_AI_MAX_SKILL_TOKENS,
  PLATFORM_AI_SKILL_POLICY_VERSION,
  type PlatformAICapability,
  type PlatformAICapabilitySnapshotV2,
} from '@/api/platformAI'

const SNAPSHOT_FIELDS = [
  'schema_version',
  'capability_key',
  'audience',
  'model_policy',
  'configuration_refs',
  'skill_runtime_policy',
] as const

const SHA256_PATTERN = /^[a-f0-9]{64}$/
const STRUCTURED_AGENT_TYPES = new Set([
  'resume_profile_extractor',
  'job_requirement_extractor',
  'candidate_match_evaluator',
])

export interface AgentSkillReleaseOption {
  value: number
  label: string
  skillId: number
  skillEnabled: boolean
  version: string
  compiledHash: string
  agentType: string
  scenario: string
  role: AgentSkillCompositionRole
  risk: AgentSkillRiskLevel
  coreEstimatedTokens: number
  packageEstimatedTokens: number
}

export interface CapabilityReleaseForm {
  allowed_llm_model_ids: number[]
  default_llm_model_id?: number
  allowed_embedding_model_ids: number[]
  default_embedding_model_id?: number
  agent_ids: number[]
  prompt_template_ids: number[]
  agent_skill_version_ids: number[]
  mcp_policy_ids: number[]
  max_skill_tokens: number
  max_input_ratio: number
  evaluation_suite_hash: string
  evaluation_result_hash: string
}

export interface CapabilityReleaseValidation {
  valid: boolean
  errors: string[]
  warnings: string[]
}

export interface EvaluationGateStatus {
  passed: boolean
  label: '通过' | '未通过'
  reason: string
}

const normalizedIDs = (ids: number[]): number[] =>
  [...new Set(ids.filter((id) => Number.isInteger(id) && id > 0))].sort((left, right) => left - right)

const normalizedHash = (value: string): string => value.trim().toLowerCase()

export const isSHA256 = (value: string): boolean => SHA256_PATTERN.test(normalizedHash(value))

export const createCapabilityReleaseForm = (): CapabilityReleaseForm => ({
  allowed_llm_model_ids: [],
  default_llm_model_id: undefined,
  allowed_embedding_model_ids: [],
  default_embedding_model_id: undefined,
  agent_ids: [],
  prompt_template_ids: [],
  agent_skill_version_ids: [],
  mcp_policy_ids: [],
  max_skill_tokens: PLATFORM_AI_MAX_SKILL_TOKENS,
  max_input_ratio: PLATFORM_AI_MAX_INPUT_RATIO,
  evaluation_suite_hash: '',
  evaluation_result_hash: '',
})

export const buildCapabilitySnapshot = (
  capability: Pick<PlatformAICapability, 'capability_key' | 'audience'>,
  form: CapabilityReleaseForm,
): PlatformAICapabilitySnapshotV2 => ({
  schema_version: PLATFORM_AI_CAPABILITY_SCHEMA_VERSION,
  capability_key: capability.capability_key,
  audience: capability.audience,
  model_policy: {
    allowed_llm_model_ids: normalizedIDs(form.allowed_llm_model_ids),
    default_llm_model_id: form.default_llm_model_id || 0,
    allowed_embedding_model_ids: normalizedIDs(form.allowed_embedding_model_ids),
    default_embedding_model_id: form.default_embedding_model_id || 0,
  },
  configuration_refs: {
    agent_ids: normalizedIDs(form.agent_ids),
    prompt_template_ids: normalizedIDs(form.prompt_template_ids),
    agent_skill_version_ids: normalizedIDs(form.agent_skill_version_ids),
    mcp_policy_ids: normalizedIDs(form.mcp_policy_ids),
  },
  skill_runtime_policy: {
    policy_version: PLATFORM_AI_SKILL_POLICY_VERSION,
    max_skill_tokens: form.max_skill_tokens,
    max_input_ratio: form.max_input_ratio,
    max_skills: PLATFORM_AI_MAX_SKILLS,
    evaluation_suite_hash: normalizedHash(form.evaluation_suite_hash),
    evaluation_result_hash: normalizedHash(form.evaluation_result_hash),
  },
})

export const serializeCapabilitySnapshot = (
  capability: Pick<PlatformAICapability, 'capability_key' | 'audience'>,
  form: CapabilityReleaseForm,
): string => JSON.stringify(buildCapabilitySnapshot(capability, form))

export const parseCapabilitySnapshot = (value: string): PlatformAICapabilitySnapshotV2 | null => {
  try {
    const parsed = JSON.parse(value) as unknown
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
      ? parsed as PlatformAICapabilitySnapshotV2
      : null
  } catch {
    return null
  }
}

export const applySnapshotToForm = (
  form: CapabilityReleaseForm,
  snapshot: PlatformAICapabilitySnapshotV2 | null,
): void => {
  const next = snapshot || buildCapabilitySnapshot(
    { capability_key: '', audience: 'tenant_hr' },
    createCapabilityReleaseForm(),
  )
  form.allowed_llm_model_ids = [...(next.model_policy?.allowed_llm_model_ids || [])]
  form.default_llm_model_id = next.model_policy?.default_llm_model_id || undefined
  form.allowed_embedding_model_ids = [...(next.model_policy?.allowed_embedding_model_ids || [])]
  form.default_embedding_model_id = next.model_policy?.default_embedding_model_id || undefined
  form.agent_ids = [...(next.configuration_refs?.agent_ids || [])]
  form.prompt_template_ids = [...(next.configuration_refs?.prompt_template_ids || [])]
  form.agent_skill_version_ids = [...(next.configuration_refs?.agent_skill_version_ids || [])]
  form.mcp_policy_ids = [...(next.configuration_refs?.mcp_policy_ids || [])]
  form.max_skill_tokens = next.skill_runtime_policy?.max_skill_tokens || PLATFORM_AI_MAX_SKILL_TOKENS
  form.max_input_ratio = next.skill_runtime_policy?.max_input_ratio || PLATFORM_AI_MAX_INPUT_RATIO
  form.evaluation_suite_hash = next.skill_runtime_policy?.evaluation_suite_hash || ''
  form.evaluation_result_hash = next.skill_runtime_policy?.evaluation_result_hash || ''
}

export const toAgentSkillReleaseOptions = (
  skill: AgentSkillInfo,
  versions: AgentSkillVersionInfo[],
): AgentSkillReleaseOption[] => versions.map((version) => ({
  value: version.id,
  label: `${skill.display_name || skill.name} · ${version.version}`,
  skillId: skill.id,
  skillEnabled: skill.is_enabled,
  version: version.version,
  compiledHash: version.package.compiled_hash,
  agentType: version.package.manifest.agent_type,
  scenario: version.package.manifest.scenario,
  role: version.package.manifest.composition.role,
  risk: version.package.manifest.risk,
  coreEstimatedTokens: version.package.core_estimated_tokens,
  packageEstimatedTokens: version.package.package_estimated_tokens,
}))

export const selectedSkillOptions = (
  snapshot: PlatformAICapabilitySnapshotV2 | null,
  options: AgentSkillReleaseOption[],
): AgentSkillReleaseOption[] => {
  const byID = new Map(options.map((option) => [option.value, option]))
  return (snapshot?.configuration_refs?.agent_skill_version_ids || [])
    .map((id) => byID.get(id))
    .filter((option): option is AgentSkillReleaseOption => Boolean(option))
}

export const validateSkillComposition = (
  selectedVersionIDs: number[],
  options: AgentSkillReleaseOption[],
): string[] => {
  const errors: string[] = []
  const byID = new Map(options.map((option) => [option.value, option]))
  const groups = new Map<string, { primary: AgentSkillReleaseOption[]; supporting: AgentSkillReleaseOption[] }>()
  const structuredPrimaryCounts = new Map<string, number>()

  for (const versionID of normalizedIDs(selectedVersionIDs)) {
    const option = byID.get(versionID)
    if (!option) {
      errors.push(`Agent Skill 版本 #${versionID} 不存在或未加载。`)
      continue
    }
    if (!option.skillEnabled) {
      errors.push(`${option.label} 所属 Skill 已停用。`)
    }
    const key = `${option.agentType}\u0000${option.scenario}`
    const group = groups.get(key) || { primary: [], supporting: [] }
    group[option.role].push(option)
    groups.set(key, group)
    if (option.role === 'primary' && STRUCTURED_AGENT_TYPES.has(option.agentType.trim())) {
      structuredPrimaryCounts.set(
        option.agentType,
        (structuredPrimaryCounts.get(option.agentType) || 0) + 1,
      )
    }
  }

  for (const [key, group] of groups) {
    const [agentType, scenario] = key.split('\u0000')
    const context = `Agent 类型 ${agentType || '-'}、场景 ${scenario || '-'}`
    if (group.primary.length > 1) {
      errors.push(`${context} 最多发布一个 Primary Skill。`)
    }
    if (group.supporting.length > 1) {
      errors.push(`${context} 最多发布一个 Supporting Skill。`)
    }
    if (group.supporting.length && group.primary.length !== 1) {
      errors.push(`${context} 的 Supporting Skill 必须与一个 Primary Skill 同时发布。`)
    }
  }
  for (const [agentType, count] of structuredPrimaryCounts) {
    if (count > 1) {
      errors.push(`结构化 Agent 类型 ${agentType} 最多发布一个 Primary Skill。`)
    }
  }
  return errors
}

export const evaluationGateStatus = (
  snapshot: PlatformAICapabilitySnapshotV2 | null,
): EvaluationGateStatus => {
  const policy = snapshot?.skill_runtime_policy
  if (!policy?.evaluation_suite_hash || !policy.evaluation_result_hash) {
    return { passed: false, label: '未通过', reason: '等待后端确定性评测写入 Suite 和 Result Hash。' }
  }
  if (!isSHA256(policy.evaluation_suite_hash) || !isSHA256(policy.evaluation_result_hash)) {
    return { passed: false, label: '未通过', reason: '评测 Hash 必须是 64 位 SHA-256 十六进制摘要。' }
  }
  return { passed: true, label: '通过', reason: '确定性评测凭证完整，发布时后端会重新校验结果。' }
}

export const validateCapabilitySnapshot = (
  snapshot: PlatformAICapabilitySnapshotV2 | null,
  skillOptions: AgentSkillReleaseOption[],
  requireEvaluation: boolean,
  expectedCapability?: Pick<PlatformAICapability, 'capability_key' | 'audience'>,
): CapabilityReleaseValidation => {
  const errors: string[] = []
  const warnings: string[] = []
  if (!snapshot) {
    return { valid: false, errors: ['发布快照不是有效 JSON 对象。'], warnings }
  }

  const actualFields = Object.keys(snapshot).sort()
  const expectedFields = [...SNAPSHOT_FIELDS].sort()
  if (actualFields.length !== expectedFields.length || actualFields.some((field, index) => field !== expectedFields[index])) {
    errors.push('发布快照必须且只能包含 schema_version、capability_key、audience、model_policy、configuration_refs、skill_runtime_policy。')
  }
  if (snapshot.schema_version !== PLATFORM_AI_CAPABILITY_SCHEMA_VERSION) {
    errors.push('发布快照 schema_version 必须为 2。')
  }
  if (expectedCapability &&
      (snapshot.capability_key !== expectedCapability.capability_key ||
       snapshot.audience !== expectedCapability.audience)) {
    errors.push('发布快照的 Capability Key 或 Audience 与当前能力不一致。')
  }
  if (snapshot.skill_runtime_policy?.policy_version !== PLATFORM_AI_SKILL_POLICY_VERSION) {
    errors.push(`Skill 策略版本必须为 ${PLATFORM_AI_SKILL_POLICY_VERSION}。`)
  }
  const allowedLLM = snapshot.model_policy?.allowed_llm_model_ids || []
  if (!allowedLLM.length) errors.push('未配置可用 LLM 模型池。')
  if (!snapshot.model_policy?.default_llm_model_id || !allowedLLM.includes(snapshot.model_policy.default_llm_model_id)) {
    errors.push('默认 LLM 不属于模型池。')
  }
  const allowedEmbedding = snapshot.model_policy?.allowed_embedding_model_ids || []
  const defaultEmbedding = snapshot.model_policy?.default_embedding_model_id || 0
  if (allowedEmbedding.length && (!defaultEmbedding || !allowedEmbedding.includes(defaultEmbedding))) {
    errors.push('配置 Embedding 模型池时必须选择池内默认模型。')
  }
  const maxTokens = snapshot.skill_runtime_policy?.max_skill_tokens
  if (!Number.isInteger(maxTokens) || maxTokens < 1 || maxTokens > PLATFORM_AI_MAX_SKILL_TOKENS) {
    errors.push(`Skill Token 上限必须是 1 到 ${PLATFORM_AI_MAX_SKILL_TOKENS} 的整数。`)
  }
  const ratio = snapshot.skill_runtime_policy?.max_input_ratio
  if (typeof ratio !== 'number' || !Number.isFinite(ratio) || ratio <= 0 || ratio > PLATFORM_AI_MAX_INPUT_RATIO) {
    errors.push(`Skill 输入占比必须大于 0 且不超过 ${PLATFORM_AI_MAX_INPUT_RATIO}。`)
  }
  if (snapshot.skill_runtime_policy?.max_skills !== PLATFORM_AI_MAX_SKILLS) {
    errors.push(`每次运行最多 Skill 数量固定为 ${PLATFORM_AI_MAX_SKILLS}。`)
  }
  if (!(snapshot.configuration_refs?.agent_ids || []).length &&
      !(snapshot.configuration_refs?.prompt_template_ids || []).length) {
    warnings.push('未固定 Agent 或 Prompt 引用。')
  }
  if (!(snapshot.configuration_refs?.agent_skill_version_ids || []).length) {
    warnings.push('未固定任何 Agent Skill 版本。')
  }
  errors.push(...validateSkillComposition(
    snapshot.configuration_refs?.agent_skill_version_ids || [],
    skillOptions,
  ))
  if (requireEvaluation) {
    const gate = evaluationGateStatus(snapshot)
    if (!gate.passed) errors.push(gate.reason)
  }
  return { valid: errors.length === 0, errors, warnings }
}

export const capabilityReleaseErrorMessage = (error: unknown): string => {
  const raw = (error as { response?: { data?: { message?: string; msg?: string } }; message?: string })
    ?.response?.data?.message
    || (error as { response?: { data?: { msg?: string } } })?.response?.data?.msg
    || (error as { message?: string })?.message
    || ''
  const mappings: Array<[RegExp, string]> = [
    [/requires exactly one Primary/i, 'Supporting Skill 必须与相同 Agent 类型和场景的 Primary Skill 一起发布。'],
    [/exceeds one Primary plus one Supporting/i, '同一 Agent 类型和场景最多发布一个 Primary 和一个 Supporting Skill。'],
    [/cannot publish more than one Primary/i, '同一结构化 Agent 类型最多发布一个 Primary Skill。'],
    [/evaluation hashes are required/i, '确定性评测尚未生成 Suite Hash 和 Result Hash。'],
    [/evaluation is unavailable/i, '确定性评测服务暂不可用，请稍后重试。'],
    [/evaluation did not pass|hashes do not match/i, '确定性评测未通过，或评测结果与当前草稿不一致。'],
    [/must exist and be enabled|disabled registry/i, '发布引用包含不存在或已停用的配置，请刷新后重新选择。'],
  ]
  return mappings.find(([pattern]) => pattern.test(raw))?.[1] || raw || 'AI 能力版本操作失败。'
}
