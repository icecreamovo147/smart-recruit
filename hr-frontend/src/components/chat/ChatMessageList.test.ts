import { describe, expect, it } from 'vitest'
import { shallowMount } from '@vue/test-utils'
import ChatMessageList from './ChatMessageList.vue'

const mountList = (retryDisabled: boolean, interactionDisabled = false) => shallowMount(ChatMessageList, {
  props: {
    messages: [{ role: 'assistant', content: '请求失败', failed: true, retryDisabled }],
    loading: false,
    streaming: false,
    sessionLoading: false,
    hasSession: true,
    renderMarkdown: (text: string) => text,
    waitingText: () => '响应中',
    interactionDisabled,
  },
  global: {
    stubs: {
      'el-button': { template: '<button @click="$emit(\'click\')"><slot/></button>' },
      'el-tag': true,
      'el-skeleton': true,
      'el-empty': true,
    },
  },
})

describe('ChatMessageList retry policy', () => {
  it('does not offer automatic retry for stable Context guard failures', () => {
    expect(mountList(true).find('.bubble__retry').exists()).toBe(false)
  })

  it('keeps the existing retry action for ordinary failures', async () => {
    const wrapper = mountList(false)
    await wrapper.get('.bubble__retry button').trigger('click')
    expect(wrapper.emitted('retry')?.[0]).toEqual([0])
  })

  it('blocks retry while AI credits are exhausted', async () => {
    const wrapper = mountList(false, true)
    await wrapper.get('.bubble__retry button').trigger('click')
    expect(wrapper.emitted('retry')).toBeUndefined()
  })
})

describe('ChatMessageList assistant waiting state', () => {
  it('uses the compact candidate-side typing indicator and hides live process details', () => {
    const wrapper = shallowMount(ChatMessageList, {
      props: {
        messages: [{
          role: 'assistant',
          content: '',
          pending: true,
          waitingText: '分析中',
          processContent: '正在查询招聘数据',
        }],
        loading: false,
        streaming: true,
        sessionLoading: false,
        hasSession: true,
        renderMarkdown: (text: string) => text,
        waitingText: (message) => message.waitingText || '',
      },
      global: {
        stubs: {
          'el-button': true,
          'el-tag': true,
          'el-skeleton': true,
          'el-empty': true,
        },
      },
    })

    expect(wrapper.get('.assistant-typing').attributes('role')).toBe('status')
    expect(wrapper.get('.assistant-typing__text').text()).toBe('分析中')
    expect(wrapper.findAll('.assistant-typing__dots span')).toHaveLength(3)
    expect(wrapper.find('.assistant-process').exists()).toBe(false)
    expect(wrapper.find('.assistant-loading-card').exists()).toBe(false)
  })

  it('keeps execution details available after the waiting state ends', () => {
    const wrapper = shallowMount(ChatMessageList, {
      props: {
        messages: [{
          role: 'assistant',
          content: '分析完成',
          pending: false,
          processContent: '已查询招聘数据',
        }],
        loading: false,
        streaming: false,
        sessionLoading: false,
        hasSession: true,
        renderMarkdown: (text: string) => text,
        waitingText: () => '',
      },
      global: {
        stubs: {
          'el-button': true,
          'el-tag': true,
          'el-skeleton': true,
          'el-empty': true,
        },
      },
    })

    expect(wrapper.get('.assistant-process').text()).toContain('已查询招聘数据')
    expect(wrapper.find('.assistant-typing').exists()).toBe(false)
  })
})

describe('ChatMessageList MCP confirmation', () => {
  it('renders explicit confirm and reject actions and emits rejection', async () => {
    const wrapper = shallowMount(ChatMessageList, {
      props: {
        messages: [{
          role: 'assistant',
          content: '',
          agentSkillSelection: {
            required: true,
            reason: '请授权候选人检索',
            candidates: [],
            confirmation_kind: 'mcp_tool',
            recommended_agent_skill_version_ids: [],
          },
        }],
        loading: false,
        streaming: false,
        sessionLoading: false,
        hasSession: true,
        renderMarkdown: (text: string) => text,
        waitingText: () => '',
      },
      global: {
        stubs: {
          'el-button': {
            props: ['disabled'],
            emits: ['click'],
            template: '<button :disabled="disabled" @click="$emit(\'click\')"><slot/></button>',
          },
          'el-tag': true,
          'el-skeleton': true,
          'el-empty': true,
        },
      },
    })

    expect(wrapper.get('.skill-confirmation__title').text()).toBe('确认执行 MCP 工具')
    expect(wrapper.get('.skill-confirmation__desc').text()).toBe('请授权候选人检索')
    const actions = wrapper.findAll('.skill-confirmation__actions button')
    expect(actions.map((button) => button.text())).toEqual(['拒绝并取消', '确认执行'])

    await actions[0].trigger('click')
    expect(wrapper.emitted('reject-skill-selection')?.[0]).toEqual([0])
  })
})

describe('ChatMessageList governed Agent Skill UX', () => {
  const candidate = {
    skill_id: 7,
    version_id: 701,
    version: '2.1.0',
    compiled_hash: 'abcdef0123456789',
    name: 'resume-review',
    display_name: '简历复核',
    reason: '命中简历复核场景',
    composition_role: 'primary' as const,
    risk: 'high' as const,
    activation_policy: 'confirm' as const,
    core_estimated_tokens: 240,
    recommended: true,
  }
  const supportingCandidate = {
    ...candidate,
    skill_id: 8,
    version_id: 702,
    version: '1.4.0',
    compiled_hash: 'fedcba9876543210',
    name: 'interview-rubric',
    display_name: '面试评估',
    reason: '补充面试评估标准',
    composition_role: 'supporting' as const,
    core_estimated_tokens: 180,
  }

  const mountGovernedList = (interactionDisabled = false) => shallowMount(ChatMessageList, {
    props: {
      messages: [{
        role: 'assistant',
        content: '',
        agentSkillSelection: {
          required: true,
          reason: '高风险 Skill 需要确认',
          candidates: [candidate, supportingCandidate],
          confirmation_kind: 'agent_skill',
          confirmation_id: 'skill-confirm-701',
          recommended_agent_skill_version_ids: [702, 701],
          expires_at: '2026-07-28T12:10:00Z',
        },
      }],
      loading: false,
      streaming: false,
      sessionLoading: false,
      hasSession: true,
      interactionDisabled,
      renderMarkdown: (text: string) => text,
      waitingText: () => '',
    },
    global: {
      stubs: {
        'el-button': {
          props: ['disabled'],
          emits: ['click'],
          template: '<button :disabled="disabled" @click="$emit(\'click\')"><slot/></button>',
        },
        'el-tag': true,
        'el-skeleton': true,
        'el-empty': true,
      },
    },
  })

  it('renders exact version governance and separate approve, reject, and cancel actions', async () => {
    const wrapper = mountGovernedList()
    expect(wrapper.get('.skill-confirmation__title').text()).toBe('确认启用 Agent Skill')
    expect(wrapper.get('.skill-confirmation__meta').text()).toContain(
      'v2.1.0 · abcdef0123 · primary · high · confirm · 240 tokens',
    )
    const exactVersions = wrapper.findAll('.skill-confirmation__option')
    expect(exactVersions).toHaveLength(2)
    expect(exactVersions.every((item) => item.element.tagName === 'DIV')).toBe(true)
    expect(exactVersions.every((item) => item.attributes('role') === 'listitem')).toBe(true)
    await exactVersions[0].trigger('click')
    expect(wrapper.emitted('confirm-skill-selection')).toBeUndefined()
    const actions = wrapper.findAll('.skill-confirmation__actions button')
    expect(actions.map((button) => button.text())).toEqual(['拒绝 Skill', '取消运行', '确认启用'])

    await actions[0].trigger('click')
    await actions[1].trigger('click')
    await actions[2].trigger('click')
    expect(wrapper.emitted('reject-skill-selection')?.[0]).toEqual([0])
    expect(wrapper.emitted('cancel-pending-run')?.[0]).toEqual([0])
    expect(wrapper.emitted('confirm-skill-selection')?.[0]).toEqual([0, [702, 701]])
  })

  it('fails closed when the server did not provide exact recommended version IDs', async () => {
    const wrapper = mountGovernedList()
    await wrapper.setProps({
      messages: [{
        role: 'assistant',
        content: '',
        agentSkillSelection: {
          required: true,
          reason: '确认信息不完整',
          candidates: [candidate],
          confirmation_kind: 'agent_skill',
          confirmation_id: 'skill-confirm-701',
          recommended_agent_skill_version_ids: [],
        },
      }],
    })
    const actions = wrapper.findAll('.skill-confirmation__actions button')
    expect(actions[2].attributes('disabled')).toBeDefined()
    await actions[2].trigger('click')
    expect(wrapper.emitted('confirm-skill-selection')).toBeUndefined()
  })

  it('blocks repeated decisions while confirmation submission is in flight', async () => {
    const wrapper = mountGovernedList(true)
    const actions = wrapper.findAll('.skill-confirmation__actions button')
    await actions[0].trigger('click')
    await actions[1].trigger('click')
    await actions[2].trigger('click')
    expect(wrapper.emitted('reject-skill-selection')).toBeUndefined()
    expect(wrapper.emitted('cancel-pending-run')).toBeUndefined()
    expect(wrapper.emitted('confirm-skill-selection')).toBeUndefined()
  })

  it('renders runtime evidence without Skill content', () => {
    const wrapper = shallowMount(ChatMessageList, {
      props: {
        messages: [{
          role: 'assistant',
          content: '分析完成',
          agent_skill_runtime_evidence: [{
            skill_id: 7,
            version_id: 701,
            version: '2.1.0',
            compiled_hash: 'abcdef0123456789',
            skill_name: 'resume-review',
            display_name: '简历复核',
            composition_role: 'primary',
            risk: 'medium',
            activation_policy: 'auto',
            selection_mode: 'automatic',
            relevance_mode: 'hybrid',
            core_estimated_tokens: 240,
            loaded_tokens: 360,
            included: true,
            decision_reason: 'selected_primary',
            sections: [{
              section_id: 1,
              section_key: 'screening-rules',
              content_hash: 'section-hash',
              estimated_tokens: 120,
              final_rank_score: 0.92,
              included: true,
              decision_reason: 'within_budget',
            }, {
              section_id: 2,
              section_key: 'long-reference',
              content_hash: 'dropped-hash',
              estimated_tokens: 900,
              final_rank_score: 0.4,
              included: false,
              decision_reason: 'budget_exceeded',
            }],
          }],
        }],
        loading: false,
        streaming: false,
        sessionLoading: false,
        hasSession: true,
        renderMarkdown: (text: string) => text,
        waitingText: () => '',
      },
      global: {
        stubs: {
          'el-button': true,
          'el-tag': true,
          'el-skeleton': true,
          'el-empty': true,
        },
      },
    })

    const evidence = wrapper.get('.agent-skill-evidence')
    expect(evidence.text()).toContain('简历复核')
    expect(evidence.text()).toContain('v2.1.0 · abcdef0123')
    expect(evidence.text()).toContain('screening-rules · 120 tokens · within_budget')
    expect(evidence.text()).toContain('long-reference · 900 tokens · budget_exceeded')
    expect(evidence.text()).not.toContain('section-hash')
  })
})
