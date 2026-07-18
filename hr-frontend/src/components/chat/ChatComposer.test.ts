import { describe, expect, it } from 'vitest'
import { shallowMount } from '@vue/test-utils'
import ChatComposer from './ChatComposer.vue'
import type { ContextUsageInfo } from '@/types/ai'

const baseUsage = (patch: Partial<ContextUsageInfo> = {}): ContextUsageInfo => ({
  model_id: 1,
  model_name: 'qwen-plus',
  context_window_tokens: 8_192,
  max_output_tokens: 1_024,
  prompt_tokens_estimated: 851,
  prompt_tokens_actual: 0,
  completion_tokens_actual: 0,
  total_tokens_actual: 0,
  remaining_tokens_estimated: 5_907,
  usage_ratio: 0.104,
  estimated: true,
  source: 'conservative_estimator',
  stage: 'post_turn',
  input_budget_tokens: 6_758,
  safety_margin_tokens: 410,
  budget_usage_ratio: 851 / 6_758,
  budget_status: 'within_budget',
  included_message_count: 12,
  omitted_message_count: 3,
  summary_applied: true,
  breakdown: {
    system_prompt_tokens: 200,
    recent_message_tokens: 300,
    summary_tokens: 80,
    memory_tokens: 0,
    current_message_tokens: 40,
    skill_tokens: 20,
    tool_result_tokens: 100,
    tool_schema_tokens: 60,
    protocol_overhead_tokens: 51,
  },
  ...patch,
})

const mountComposer = (contextUsage: ContextUsageInfo | null, contextPreviewing = false) => shallowMount(ChatComposer, {
  props: {
    input: '',
    loading: false,
    streaming: false,
    modelList: [],
    selectedModelId: null,
    contextUsage,
    contextPreviewing,
    dataSource: '招聘业务数据库',
    currentSession: null,
    skillCapabilities: [],
    selectedSkillKeys: [],
    agentSkills: [],
    selectedAgentSkillIds: [],
  },
  global: {
    stubs: {
      'el-popover': {
        template: '<div><slot name="reference"/><div class="test-popover"><slot/></div></div>',
      },
      'el-input': true,
      'el-select': true,
      'el-option': true,
      'el-button': true,
      'el-icon': true,
      Transition: false,
    },
  },
})

describe('ChatComposer Context indicator', () => {
  it('renders current effective input with the total context window and full details', () => {
    const wrapper = mountComposer(baseUsage())
    expect(wrapper.get('.chat-composer__context-value').text()).toBe('851 / 8.2K')
    expect(wrapper.get('.chat-composer__context-indicator').attributes('aria-label')).toContain('10.4% 已用')
    expect(wrapper.text()).toContain('上下文窗口')
    expect(wrapper.text()).toContain('当前会话已用')
    expect(wrapper.text()).toContain('安全预算占用')
    expect(wrapper.text()).toContain('总上下文窗口')
    expect(wrapper.text()).toContain('8.2K')
    expect(wrapper.text()).toContain('安全余量')
    expect(wrapper.text()).toContain('工具定义')
    expect(wrapper.text()).toContain('协议开销')
    expect(wrapper.text()).toContain('纳入 12 条消息')
    expect(wrapper.text()).toContain('省略 3 条消息')
    expect(wrapper.text()).toContain('已应用会话摘要')
    expect(wrapper.text()).toContain('当前会话')
  })

  it('prefers actual provider prompt tokens', () => {
    const wrapper = mountComposer(baseUsage({ prompt_tokens_actual: 777, source: 'provider_actual' }))
    expect(wrapper.get('.chat-composer__context-value').text()).toBe('777 / 8.2K')
    expect(wrapper.text()).toContain('Provider 实际值')
  })

  it('shows unknown budget without a fake percentage', () => {
    const wrapper = mountComposer(baseUsage({
      context_window_tokens: 0,
      input_budget_tokens: 0,
      budget_usage_ratio: 0,
      budget_status: 'unknown_config',
    }))
    expect(wrapper.get('.chat-composer__context-value').text()).toBe('851 / —')
    expect(wrapper.text()).toContain('模型上下文窗口未配置')
    expect(wrapper.text()).toContain('无法计算')
    expect(wrapper.text()).not.toContain('0.0%')
  })

  it('renders a neutral placeholder when the session has no snapshot', () => {
    const wrapper = mountComposer(null)
    expect(wrapper.get('.chat-composer__context-unknown').text()).toBe('— / —')
    expect(wrapper.get('.chat-composer__context-indicator').attributes('aria-label')).toBe('Context — / —')
  })

  it('shows the newly selected model window while context is recalculating', () => {
    const wrapper = shallowMount(ChatComposer, {
      props: {
        input: '下一步', loading: false, streaming: false,
        modelList: [{ id: 2, provider_id: 1, model_name: 'large', display_name: 'Large', temperature: 0, top_p: 1, max_tokens: 4096, context_window_tokens: 128_000, max_concurrency: 1, timeout_seconds: 60, is_enabled: true, is_default: false, created_at: '', updated_at: '', provider_name: 'test' }],
        selectedModelId: 2, contextUsage: baseUsage(), contextPreviewing: true,
        dataSource: '招聘业务数据库', currentSession: null, skillCapabilities: [], selectedSkillKeys: [], agentSkills: [], selectedAgentSkillIds: [],
      },
      global: { stubs: { 'el-popover': { template: '<div><slot name="reference"/></div>' }, 'el-input': true, 'el-select': true, 'el-option': true, 'el-button': true, 'el-icon': true, Transition: false } },
    })
    expect(wrapper.get('.chat-composer__context-unknown').text()).toBe('计算中 / 128K')
    expect(wrapper.get('.chat-composer__context-indicator').attributes('aria-label')).toBe('Context 计算中 / 128K')
  })
})
