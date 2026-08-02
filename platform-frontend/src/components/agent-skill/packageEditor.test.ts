import { describe, expect, it } from 'vitest'
import {
  activationPolicyForRisk,
  buildAgentSkillPackageDraft,
  createEmptyPackageEditorModel,
  createNextEmptySection,
  createEmptySection,
  validatePackageEditor,
} from './packageEditor'

const validModel = () => {
  const model = createEmptyPackageEditorModel()
  model.skillName = 'candidate_job_match'
  model.displayName = '候选人与岗位匹配'
  model.description = '基于岗位要求和候选人证据给出匹配建议。'
  model.category = 'candidate_match'
  model.scenario = 'candidate_job_match'
  model.triggerKeywords = ['岗位匹配', ' 岗位匹配 ', '候选人']
  model.coreMarkdown = '# Core\n\n仅引用输入中存在的招聘事实。'
  const section = createEmptySection(0)
  section.sectionKey = 'evidence_rules'
  section.title = '证据规则'
  section.contentMarkdown = '逐条列出岗位要求对应的简历证据。'
  section.triggerTerms = ['证据']
  model.sections = [section]
  return model
}

describe('Agent Skill package editor model', () => {
  it('builds a package-only payload without client compiled authority', () => {
    const model = validModel()
    const draft = buildAgentSkillPackageDraft(model)

    expect(draft.manifest).toMatchObject({
      schema_version: 2,
      skill_name: 'candidate_job_match',
      activation_policy: 'auto',
      composition: { role: 'primary' },
      trigger_keywords: ['岗位匹配', '候选人'],
    })
    expect(draft.sections).toEqual([
      expect.objectContaining({
        section_key: 'evidence_rules',
        ordinal: 0,
        trigger_terms: ['证据'],
      }),
    ])
    expect(draft.authoring_json).toContain('platform-package-v2')
    expect(draft).not.toHaveProperty('compiled_hash')
    expect(draft).not.toHaveProperty('estimated_tokens')
    expect(draft).not.toHaveProperty('skill_md')
    expect(draft).not.toHaveProperty('flow_json')
  })

  it('derives activation policy exclusively from risk', () => {
    expect(activationPolicyForRisk('low')).toBe('auto')
    expect(activationPolicyForRisk('medium')).toBe('auto')
    expect(activationPolicyForRisk('high')).toBe('confirm')
    expect(activationPolicyForRisk('critical')).toBe('manual_only')
  })

  it('rejects supporting output contracts and strict chat output', () => {
    const supporting = validModel()
    supporting.compositionRole = 'supporting'
    supporting.outputMode = 'advisory'
    expect(validatePackageEditor(supporting).errors).toContain('Supporting Skill 不能声明输出契约。')

    const chatStrict = validModel()
    chatStrict.outputMode = 'strict'
    chatStrict.outputSchemaId = 'candidate.match.v1'
    chatStrict.outputSchemaJson = '{"type":"object"}'
    expect(validatePackageEditor(chatStrict).errors).toContain('Strict 输出只适用于三个结构化招聘运行时。')
  })

  it('accepts a strict package for a registered structured runtime', () => {
    const model = validModel()
    model.agentType = 'candidate_match_evaluator'
    model.outputMode = 'strict'
    model.outputSchemaId = 'candidate.match.v1'
    model.outputSchemaJson = '{"type":"object"}'

    expect(validatePackageEditor(model)).toMatchObject({ valid: true, errors: [] })
  })

  it('requires advisory and strict schemas to be supplied JSON objects', () => {
    const advisory = validModel()
    advisory.outputSchemaJson = ''
    expect(validatePackageEditor(advisory).errors).toContain(
      'Advisory 和 Strict 输出必须填写 JSON Schema。',
    )

    for (const schemaJSON of ['null', '[]', '"text"']) {
      advisory.outputSchemaJson = schemaJSON
      expect(validatePackageEditor(advisory).errors).toContain(
        '输出 Schema 必须是 JSON 对象。',
      )
    }

    advisory.outputSchemaJson = '{}'
    expect(validatePackageEditor(advisory).errors).not.toContain(
      '输出 Schema 必须是 JSON 对象。',
    )

    const strict = validModel()
    strict.agentType = 'candidate_match_evaluator'
    strict.outputMode = 'strict'
    strict.outputSchemaId = 'candidate.match.v1'
    strict.outputSchemaJson = ''
    expect(validatePackageEditor(strict).errors).toContain(
      'Advisory 和 Strict 输出必须填写 JSON Schema。',
    )
  })

  it('chooses the first unused generated section key', () => {
    const first = createEmptySection(0)
    const third = createEmptySection(2)
    expect(createNextEmptySection([third, first])).toMatchObject({
      sectionKey: 'reference_2',
      title: '参考资料 2',
    })
  })

  it('preserves the current section order in payload ordinals and authoring data', () => {
    const model = validModel()
    const later = createEmptySection(1)
    later.title = '补充规则'
    later.contentMarkdown = '补充证据规则。'
    later.triggerTerms = ['补充']
    model.sections = [later, model.sections[0]]

    const draft = buildAgentSkillPackageDraft(model)
    expect((draft.sections ?? []).map((section) => [section.section_key, section.ordinal])).toEqual([
      ['reference_2', 0],
      ['evidence_rules', 1],
    ])
    expect(JSON.parse(draft.authoring_json ?? '{}').section_order).toEqual([
      'reference_2',
      'evidence_rules',
    ])
  })
})
