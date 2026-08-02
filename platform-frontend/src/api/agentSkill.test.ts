import { beforeEach, describe, expect, it, vi } from 'vitest'

const request = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  patch: vi.fn(),
}))

vi.mock('./http', () => ({ default: request }))
vi.mock('@shared/utils/debugLog', () => ({
  debugLog: {
    skill: {
      info: vi.fn(),
      warn: vi.fn(),
      error: vi.fn(),
    },
  },
}))

import {
  createAgentSkill,
  createAgentSkillVersion,
  debugSemanticRetrieval,
  previewAgentSkill,
  regenerateAgentSkillVersionEmbedding,
} from './agentSkill'
import type { AgentSkillPackageDraft, AgentSkillPackageInfo } from '@shared/types/agentSkill'

const packageDraft: AgentSkillPackageDraft = {
  manifest: {
    schema_version: 2,
    skill_name: 'candidate_job_match',
    display_name: '候选人与岗位匹配',
    agent_type: 'hr_recruiting_agent',
    risk: 'medium',
    composition: { role: 'primary' },
    output_contract: { mode: 'advisory', schema_id: '', schema_json: '' },
  },
  core_markdown: '只引用已有招聘事实。',
  sections: [],
  authoring_json: '{"editor":"platform-package-v2"}',
}

const packageInfo = {
  manifest: {
    ...packageDraft.manifest,
    description: '',
    category: 'general',
    scenario: '',
    priority: 0,
    activation_policy: 'auto',
    required_capabilities: [],
    trigger_keywords: [],
    semantic_tags: [],
    output_contract: { mode: 'advisory', schema_id: '', schema_json: '' },
    evaluation_criteria: [],
  },
  core_markdown: packageDraft.core_markdown,
  sections: [],
  compiled_markdown: '# compiled',
  authoring_json: packageDraft.authoring_json || '',
  compiled_hash: 'abc123',
  core_estimated_tokens: 12,
  package_estimated_tokens: 12,
} satisfies AgentSkillPackageInfo

describe('Agent Skill Package v2 API', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('previews a package draft and returns server compilation evidence', async () => {
    request.post.mockResolvedValueOnce({ package: packageInfo })

    const result = await previewAgentSkill({ package: packageDraft })

    expect(request.post).toHaveBeenCalledWith(
      '/api/v1/platform/ai/agent-skills/preview',
      { package: packageDraft },
    )
    expect(result.package).toMatchObject({
      compiled_hash: 'abc123',
      core_estimated_tokens: 12,
      package_estimated_tokens: 12,
    })
  })

  it('creates an immutable version using only the package draft', async () => {
    request.post.mockResolvedValueOnce({
      version: {
        id: 32,
        skill_id: 7,
        version: '1.1.0',
        change_note: 'add evidence rules',
        created_at: '',
        package: packageInfo,
      },
    })
    const payload = {
      version: '1.1.0',
      package: packageDraft,
      change_note: 'add evidence rules',
      activate: true,
    }

    await createAgentSkillVersion(7, payload)

    expect(request.post).toHaveBeenCalledWith(
      '/api/v1/platform/ai/agent-skills/7/versions',
      payload,
    )
    expect(payload).not.toHaveProperty('skill_md')
    expect(payload).not.toHaveProperty('flow_json')
  })

  it('creates a registry and first version without legacy runtime fields', async () => {
    request.post.mockResolvedValueOnce({
      skill: {
        id: 7,
        name: 'candidate_job_match',
        display_name: '候选人与岗位匹配',
        description: '',
        current_version_id: 32,
        is_enabled: true,
        is_manual_invocable: true,
        created_at: '',
        updated_at: '',
        current_version: null,
      },
    })
    const payload = {
      version: '1.0.0',
      package: packageDraft,
      is_enabled: true,
      is_enabled_set: true,
      is_manual_invocable: true,
      is_manual_invocable_set: true,
      activate: true,
    }

    await createAgentSkill(payload)

    expect(request.post).toHaveBeenCalledWith('/api/v1/platform/ai/agent-skills', payload)
    expect(payload).not.toHaveProperty('agent_type')
    expect(payload).not.toHaveProperty('risk_level')
    expect(payload).not.toHaveProperty('output_schema')
    expect(payload).not.toHaveProperty('skill_md')
    expect(payload).not.toHaveProperty('flow_json')
  })

  it('regenerates embeddings by exact current version id', async () => {
    request.post.mockResolvedValueOnce({ success_count: 2, failed_count: 0, skipped_count: 0 })

    await regenerateAgentSkillVersionEmbedding(32)

    expect(request.post).toHaveBeenCalledWith(
      '/api/v1/platform/ai/agent-skills/32/embedding/regenerate',
    )
  })

  it('sends an explicit tenant-scoped HR owner context for semantic retrieval', async () => {
    request.get.mockResolvedValueOnce({ skills: [], memories: [], embedding_available: true })

    await debugSemanticRetrieval({
      query: '候选人与岗位是否匹配',
      agent_type: 'hr_recruiting_agent',
      tenant_id: 7,
      owner_role: 2,
      owner_id: 41,
      limit: 5,
    })

    expect(request.get).toHaveBeenCalledWith(
      '/api/v1/platform/ai/agent-skills/semantic-debug',
      {
        params: {
          query: '候选人与岗位是否匹配',
          agent_type: 'hr_recruiting_agent',
          tenant_id: 7,
          limit: 5,
          owner_role: 2,
          owner_id: 41,
        },
      },
    )
  })
})
