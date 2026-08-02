import type {
  AgentSkillActivationPolicy,
  AgentSkillCompositionRole,
  AgentSkillOutputMode,
  AgentSkillPackageDraft,
  AgentSkillPackageInfo,
  AgentSkillRiskLevel,
  AgentSkillSectionDraft,
} from '@shared/types/agentSkill'

export interface AgentSkillSectionEditorModel {
  sectionKey: string
  title: string
  description: string
  contentMarkdown: string
  triggerTerms: string[]
  semanticTags: string[]
  plannerIntents: string[]
  priority: number
}

export interface AgentSkillPackageEditorModel {
  skillName: string
  displayName: string
  description: string
  agentType: string
  category: string
  scenario: string
  priority: number
  risk: AgentSkillRiskLevel
  requiredCapabilities: string[]
  triggerKeywords: string[]
  semanticTags: string[]
  compositionRole: AgentSkillCompositionRole
  outputMode: AgentSkillOutputMode
  outputSchemaId: string
  outputSchemaJson: string
  evaluationCriteria: string[]
  coreMarkdown: string
  sections: AgentSkillSectionEditorModel[]
}

export interface AgentSkillEditorValidation {
  valid: boolean
  errors: string[]
  warnings: string[]
}

const STRICT_AGENT_TYPES = new Set([
  'resume_profile_extractor',
  'job_requirement_extractor',
  'candidate_match_evaluator',
])

export const activationPolicyForRisk = (
  risk: AgentSkillRiskLevel,
): AgentSkillActivationPolicy => {
  if (risk === 'critical') return 'manual_only'
  if (risk === 'high') return 'confirm'
  return 'auto'
}

export const createEmptyPackageEditorModel = (): AgentSkillPackageEditorModel => ({
  skillName: '',
  displayName: '',
  description: '',
  agentType: 'hr_recruiting_agent',
  category: 'general',
  scenario: '',
  priority: 0,
  risk: 'medium',
  requiredCapabilities: [],
  triggerKeywords: [],
  semanticTags: [],
  compositionRole: 'primary',
  outputMode: 'advisory',
  outputSchemaId: '',
  outputSchemaJson: '{"type":"object"}',
  evaluationCriteria: [],
  coreMarkdown: '',
  sections: [],
})

export const createEmptySection = (index: number): AgentSkillSectionEditorModel => ({
  sectionKey: `reference_${index + 1}`,
  title: `参考资料 ${index + 1}`,
  description: '',
  contentMarkdown: '',
  triggerTerms: [],
  semanticTags: [],
  plannerIntents: [],
  priority: 0,
})

export const createNextEmptySection = (
  sections: AgentSkillSectionEditorModel[],
): AgentSkillSectionEditorModel => {
  const usedKeys = new Set(sections.map((section) => section.sectionKey.trim()))
  let referenceNumber = 1
  while (usedKeys.has(`reference_${referenceNumber}`)) referenceNumber += 1
  return createEmptySection(referenceNumber - 1)
}

const uniqueValues = (values: string[]) => Array.from(new Set(
  values.map((value) => value.trim()).filter(Boolean),
))

const normalizedOutputContract = (model: AgentSkillPackageEditorModel) => {
  if (model.compositionRole === 'supporting' || model.outputMode === 'none') {
    return { mode: 'none' as const, schema_id: '', schema_json: '' }
  }
  return {
    mode: model.outputMode,
    schema_id: model.outputSchemaId.trim(),
    schema_json: model.outputSchemaJson.trim(),
  }
}

export const buildAgentSkillPackageDraft = (
  model: AgentSkillPackageEditorModel,
): AgentSkillPackageDraft => ({
  manifest: {
    schema_version: 2,
    skill_name: model.skillName.trim(),
    display_name: model.displayName.trim(),
    description: model.description.trim(),
    agent_type: model.agentType.trim(),
    category: model.category.trim(),
    scenario: model.scenario.trim(),
    priority: Number(model.priority) || 0,
    risk: model.risk,
    activation_policy: activationPolicyForRisk(model.risk),
    required_capabilities: uniqueValues(model.requiredCapabilities),
    trigger_keywords: uniqueValues(model.triggerKeywords),
    semantic_tags: uniqueValues(model.semanticTags),
    composition: { role: model.compositionRole },
    output_contract: normalizedOutputContract(model),
    evaluation_criteria: uniqueValues(model.evaluationCriteria),
  },
  core_markdown: model.coreMarkdown.trim(),
  sections: model.sections.map<AgentSkillSectionDraft>((section, ordinal) => ({
    section_key: section.sectionKey.trim(),
    title: section.title.trim(),
    description: section.description.trim(),
    content_markdown: section.contentMarkdown.trim(),
    trigger_terms: uniqueValues(section.triggerTerms),
    semantic_tags: uniqueValues(section.semanticTags),
    planner_intents: uniqueValues(section.plannerIntents),
    priority: Number(section.priority) || 0,
    ordinal,
  })),
  authoring_json: JSON.stringify({
    editor: 'platform-package-v2',
    section_order: model.sections.map((section) => section.sectionKey.trim()),
  }),
})

export const validatePackageEditor = (
  model: AgentSkillPackageEditorModel,
): AgentSkillEditorValidation => {
  const errors: string[] = []
  const warnings: string[] = []
  const skillName = model.skillName.trim()

  if (!/^[a-z][a-z0-9_-]{1,127}$/.test(skillName)) {
    errors.push('唯一标识必须以小写字母开头，只能包含小写字母、数字、下划线或短横线，长度为 2-128。')
  }
  if (!model.displayName.trim()) errors.push('请填写显示名称。')
  if (!model.description.trim()) errors.push('请填写 Skill 描述。')
  if (!model.agentType.trim()) errors.push('请选择适用 Agent。')
  if (!model.category.trim()) errors.push('请填写治理分类。')
  if (model.priority < -1000 || model.priority > 1000) errors.push('优先级必须在 -1000 到 1000 之间。')
  if (!model.coreMarkdown.trim()) errors.push('请填写 Core 指令。')
  if (model.sections.length > 20) errors.push('一个版本最多包含 20 个 Reference Section。')

  if (model.compositionRole === 'supporting' && model.outputMode !== 'none') {
    errors.push('Supporting Skill 不能声明输出契约。')
  }
  if (model.outputMode === 'strict' && !STRICT_AGENT_TYPES.has(model.agentType)) {
    errors.push('Strict 输出只适用于三个结构化招聘运行时。')
  }
  if (model.outputMode === 'strict' && !model.outputSchemaId.trim()) {
    errors.push('Strict 输出必须填写 Schema ID。')
  }
  if (model.outputMode !== 'none') {
    const schemaJSON = model.outputSchemaJson.trim()
    if (!schemaJSON) {
      errors.push('Advisory 和 Strict 输出必须填写 JSON Schema。')
    } else {
      try {
        const schema = JSON.parse(schemaJSON)
        if (typeof schema !== 'object' || schema === null || Array.isArray(schema)) {
          errors.push('输出 Schema 必须是 JSON 对象。')
        }
      } catch {
        errors.push('输出 Schema 必须是合法 JSON。')
      }
    }
  }

  const sectionKeys = new Set<string>()
  model.sections.forEach((section, index) => {
    const label = `Reference Section ${index + 1}`
    const sectionKey = section.sectionKey.trim()
    if (!/^[a-z][a-z0-9_-]{1,127}$/.test(sectionKey)) {
      errors.push(`${label} 的 Key 格式不正确。`)
    } else if (sectionKeys.has(sectionKey)) {
      errors.push(`${label} 的 Key 与其他 Section 重复。`)
    }
    sectionKeys.add(sectionKey)
    if (!section.title.trim()) errors.push(`${label} 必须填写标题。`)
    if (!section.contentMarkdown.trim()) errors.push(`${label} 必须填写 Markdown 内容。`)
    if (!section.triggerTerms.some((value) => value.trim())
      && !section.semanticTags.some((value) => value.trim())
      && !section.plannerIntents.some((value) => value.trim())) {
      warnings.push(`${label} 未配置召回信号，可能无法按需加载。`)
    }
  })

  return { valid: errors.length === 0, errors, warnings }
}

export const packageInfoToEditorModel = (
  packageInfo: AgentSkillPackageInfo,
): AgentSkillPackageEditorModel => ({
  skillName: packageInfo.manifest.skill_name,
  displayName: packageInfo.manifest.display_name,
  description: packageInfo.manifest.description,
  agentType: packageInfo.manifest.agent_type,
  category: packageInfo.manifest.category,
  scenario: packageInfo.manifest.scenario,
  priority: packageInfo.manifest.priority,
  risk: packageInfo.manifest.risk,
  requiredCapabilities: [...packageInfo.manifest.required_capabilities],
  triggerKeywords: [...packageInfo.manifest.trigger_keywords],
  semanticTags: [...packageInfo.manifest.semantic_tags],
  compositionRole: packageInfo.manifest.composition.role,
  outputMode: packageInfo.manifest.output_contract.mode,
  outputSchemaId: packageInfo.manifest.output_contract.schema_id,
  outputSchemaJson: packageInfo.manifest.output_contract.schema_json,
  evaluationCriteria: [...packageInfo.manifest.evaluation_criteria],
  coreMarkdown: packageInfo.core_markdown,
  sections: packageInfo.sections
    .slice()
    .sort((left, right) => left.ordinal - right.ordinal)
    .map((section) => ({
      sectionKey: section.section_key,
      title: section.title,
      description: section.description,
      contentMarkdown: section.content_markdown,
      triggerTerms: [...section.trigger_terms],
      semanticTags: [...section.semantic_tags],
      plannerIntents: [...section.planner_intents],
      priority: section.priority,
    })),
})
