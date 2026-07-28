import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { PLATFORM_PERMISSIONS } from '@/permissions'
import { useAuthStore } from '@/stores/auth'
import type { PlatformAICapabilityVersion } from '@/api/platformAI'
import AICapabilityReleaseView from './AICapabilityReleaseView.vue'

const platformAIMocks = vi.hoisted(() => ({
  createDraft: vi.fn(),
  deleteDraft: vi.fn(),
  listCapabilities: vi.fn(),
  listVersions: vi.fn(),
  publishVersion: vi.fn(),
  updateDraft: vi.fn(),
}))

vi.mock('@/api/platformAI', async (importOriginal) => ({
  ...await importOriginal<typeof import('@/api/platformAI')>(),
  createPlatformAICapabilityDraft: platformAIMocks.createDraft,
  deletePlatformAICapabilityDraft: platformAIMocks.deleteDraft,
  listPlatformAICapabilities: platformAIMocks.listCapabilities,
  listPlatformAICapabilityVersions: platformAIMocks.listVersions,
  publishPlatformAICapabilityVersion: platformAIMocks.publishVersion,
  updatePlatformAICapabilityDraft: platformAIMocks.updateDraft,
}))

vi.mock('@/api/llm', () => ({
  listModels: vi.fn().mockResolvedValue({
    list: [{ id: 9, display_name: 'Test LLM', model_name: 'test-llm', is_enabled: true }],
  }),
}))
vi.mock('@/api/embedding', () => ({
  listEmbeddingModels: vi.fn().mockResolvedValue({ list: [] }),
}))
vi.mock('@/api/agent', () => ({
  listAgentConfigs: vi.fn().mockResolvedValue({ list: [] }),
}))
vi.mock('@/api/prompt', () => ({
  listPromptTemplates: vi.fn().mockResolvedValue({ list: [] }),
}))
vi.mock('@/api/agentSkill', () => ({
  listAgentSkills: vi.fn().mockResolvedValue({ list: [] }),
  listAgentSkillVersions: vi.fn().mockResolvedValue({ list: [] }),
}))
vi.mock('@/api/mcp', () => ({
  listMcpToolPolicies: vi.fn().mockResolvedValue({ list: [] }),
}))

const capability = {
  id: 7,
  capability_key: 'ai.agent_run',
  audience: 'tenant_hr' as const,
  name: 'HR Agent',
  description: 'test',
  status: 'active',
  current_published_version_id: 51,
}

const snapshotJSON = JSON.stringify({
  schema_version: 2,
  capability_key: capability.capability_key,
  audience: capability.audience,
  model_policy: {
    allowed_llm_model_ids: [9],
    default_llm_model_id: 9,
    allowed_embedding_model_ids: [],
    default_embedding_model_id: 0,
  },
  configuration_refs: {
    agent_ids: [20],
    prompt_template_ids: [],
    agent_skill_version_ids: [],
    mcp_policy_ids: [],
  },
  skill_runtime_policy: {
    policy_version: 'skill-package-v2.1',
    max_skill_tokens: 3000,
    max_input_ratio: 0.15,
    max_skills: 2,
    evaluation_suite_hash: 'a'.repeat(64),
    evaluation_result_hash: 'b'.repeat(64),
  },
})

const version = (
  id: number,
  versionNumber: number,
  changeNote: string,
  status: PlatformAICapabilityVersion['status'],
): PlatformAICapabilityVersion => ({
  id,
  capability_id: capability.id,
  version: versionNumber,
  status,
  snapshot_json: snapshotJSON,
  snapshot_hash: 'c'.repeat(64),
  change_note: changeNote,
  published_at: '',
})

const deferred = <T>() => {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((next) => { resolve = next })
  return { promise, resolve }
}

describe('AICapabilityReleaseView draft submission', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setActivePinia(createPinia())
    useAuthStore().persist({
      user_id: 1,
      username: 'operator',
      account_type: 'platform',
      roles: ['platform_operator'],
      permissions: [PLATFORM_PERMISSIONS.AI_RELEASE_MANAGE],
      client_app: 'platform',
      available_apps: ['platform'],
    })
    platformAIMocks.listCapabilities.mockResolvedValue({ list: [capability] })
    platformAIMocks.listVersions.mockResolvedValue({
      list: [version(51, 1, 'published baseline', 'published')],
    })
  })

  it('keeps draft save single-flight during a deferred double click and uses the server version', async () => {
    const saveResult = deferred<{ version: PlatformAICapabilityVersion }>()
    platformAIMocks.createDraft.mockReturnValue(saveResult.promise)
    const wrapper = mount(AICapabilityReleaseView, {
      global: {
        plugins: [ElementPlus],
        stubs: {
          ElDialog: {
            props: ['modelValue'],
            template: '<div v-if="modelValue"><slot /><footer><slot name="footer" /></footer></div>',
          },
          PageHeader: { template: '<header><slot name="primary" /></header>' },
          PagePanel: { template: '<div><slot /></div>' },
        },
      },
    })
    await flushPromises()

    const createButton = wrapper.findAll('button')
      .find((button) => button.text().includes('创建版本草稿'))
    expect(createButton).toBeDefined()
    await createButton!.trigger('click')
    await flushPromises()

    await wrapper.get('.editor-form .el-input__inner').setValue('server evaluated draft')
    const saveButton = wrapper.get('[data-testid="save-capability-draft"]')
    const firstClick = saveButton.trigger('click')
    const secondClick = saveButton.trigger('click')
    await nextTick()

    expect(platformAIMocks.createDraft).toHaveBeenCalledTimes(1)
    expect(saveButton.attributes('disabled')).toBeDefined()
    expect(saveButton.classes()).toContain('is-loading')

    saveResult.resolve({
      version: version(72, 2, 'server evaluated draft', 'draft'),
    })
    await Promise.all([firstClick, secondClick])
    await flushPromises()

    expect(platformAIMocks.listVersions).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('V2')
    expect(wrapper.text()).toContain('server evaluated draft')
    wrapper.unmount()
  })
})
