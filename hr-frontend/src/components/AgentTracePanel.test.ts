import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import type { AgentRunItem, ToolTraceItem } from '@/types/ai'
import AgentTracePanel from './AgentTracePanel.vue'

const { getAgentRuns, getToolTraces, getActiveAgentRun, subscribeAgentRunEvents } = vi.hoisted(() => ({
  getAgentRuns: vi.fn(),
  getToolTraces: vi.fn(),
  getActiveAgentRun: vi.fn(),
  subscribeAgentRunEvents: vi.fn(),
}))

vi.mock('@/api/ai', () => ({
  getAgentRuns,
  getToolTraces,
}))

vi.mock('@/api/agentRun', () => ({
  getActiveAgentRun,
  subscribeAgentRunEvents,
  // ensure cancel is not used by panel module graph in this test
  cancelAgentRun: vi.fn(),
}))

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<typeof import('element-plus')>('element-plus')
  return {
    ...actual,
    ElMessage: { error: vi.fn(), success: vi.fn(), warning: vi.fn() },
  }
})

function stubEl(name: string) {
  return defineComponent({
    name,
    props: ['modelValue', 'size', 'type', 'title', 'description', 'closable', 'effect', 'timestamp', 'color'],
    emits: ['update:modelValue'],
    setup(props, { slots, attrs }) {
      return () => h(
        'div',
        { class: name, ...attrs, 'data-size': props.size as string | undefined },
        [
          props.title ? h('span', { class: `${name}__title` }, String(props.title)) : null,
          props.description ? h('span', { class: `${name}__description` }, String(props.description)) : null,
          slots.default?.(),
        ],
      )
    },
  })
}

async function openPanel(sessionId: number) {
  const wrapper = mountPanel({ sessionId, visible: false })
  await wrapper.setProps({ visible: true })
  // allow loadTraces promises to settle
  await Promise.resolve()
  await Promise.resolve()
  await nextTick()
  await nextTick()
  return wrapper
}

function makeRun(partial: Partial<AgentRunItem> = {}): AgentRunItem {
  return {
    id: 10,
    session_id: 1,
    message_id: 2,
    history_id: 3,
    hr_id: 4,
    agent_type: 'hr_recruiting_agent',
    agent_id: 5,
    agent_name: 'hr_recruiting_agent',
    model_id: 6,
    model_name: 'qwen-plus',
    status: 'failed',
    plan_json: JSON.stringify({
      runtime: 'adk',
      risk_flags: ['verify_candidate_identity'],
      decision: { intent: 'candidate_match_evaluation', failed: true },
      recruiting_plan: { intent: 'candidate_match_evaluation' },
    }),
    final_answer: '**hello** answer',
    error_type: 'tool_error',
    error_message: 'run failed',
    started_at: '2026-07-11T10:00:00Z',
    completed_at: '2026-07-11T10:00:05Z',
    created_at: '2026-07-11T10:00:00Z',
    steps: [
      {
        id: 1,
        run_id: 10,
        step_index: 0,
        step_type: 'tool',
        capability_source: '',
        capability_key: '',
        tool_name: 'search_candidates',
        input_json: '{"q":1}',
        output_json: JSON.stringify({
          policy_decision: 'deny',
          policy_reason: 'blocked',
          policy_id: 1,
        }),
        status: 'failed',
        duration_ms: 12,
        error_message: 'step boom',
        started_at: '2026-07-11T10:00:00Z',
        completed_at: '2026-07-11T10:00:01Z',
        created_at: '2026-07-11T10:00:00Z',
      },
    ],
    ...partial,
  }
}

function makeTrace(partial: Partial<ToolTraceItem> = {}): ToolTraceItem {
  return {
    id: 100,
    session_id: 1,
    tool_name: 'legacy_tool',
    args_json: '{"a":1}',
    result_content: '{"ok":true}',
    duration_ms: 5,
    error_msg: '',
    created_at: '2026-07-11T09:00:00Z',
    ...partial,
  }
}

function mountPanel(props: { sessionId: number | null; visible: boolean }) {
  return mount(AgentTracePanel, {
    props,
    global: {
      stubs: {
        'el-drawer': stubEl('el-drawer'),
        'el-empty': stubEl('el-empty'),
        'el-tag': stubEl('el-tag'),
        'el-alert': stubEl('el-alert'),
        'el-timeline': stubEl('el-timeline'),
        'el-timeline-item': stubEl('el-timeline-item'),
        'el-tabs': stubEl('el-tabs'),
        'el-tab-pane': defineComponent({
          name: 'el-tab-pane',
          props: ['label', 'name'],
          setup(_, { slots }) {
            return () => h('div', { class: 'el-tab-pane' }, slots.default?.())
          },
        }),
        'el-input': stubEl('el-input'),
        'el-select': stubEl('el-select'),
        'el-option': stubEl('el-option'),
        'el-checkbox': stubEl('el-checkbox'),
        'el-button': stubEl('el-button'),
      },
      directives: {
        loading: () => {},
      },
    },
  })
}

describe('AgentTracePanel overview shell', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getActiveAgentRun.mockResolvedValue({ run: null, has_active_run: false })
    subscribeAgentRunEvents.mockResolvedValue(undefined)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('shows overview and issue summary before timeline for durable runs', async () => {
    getAgentRuns.mockResolvedValue({ list: [makeRun()] })
    getToolTraces.mockResolvedValue({ list: [] })

    const wrapper = await openPanel(1)

    expect(wrapper.find('[data-testid="trace-overview"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="trace-issue-summary"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('执行概览')
    expect(wrapper.text()).toContain('异常与风险摘要')
    expect(wrapper.text()).toContain('search_candidates')
    // final answer markdown sanitized path still renders content
    expect(wrapper.html()).toContain('hello')
    expect(wrapper.html()).toContain('min(860px')
  })

  it('keeps legacy-only sessions usable with overview', async () => {
    getAgentRuns.mockResolvedValue({ list: [] })
    getToolTraces.mockResolvedValue({
      list: [makeTrace({ error_msg: 'legacy fail', tool_name: 'old_tool' })],
    })

    const wrapper = await openPanel(2)

    expect(wrapper.find('[data-testid="trace-overview"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('兼容工具调用明细')
    expect(wrapper.text()).toContain('old_tool')
    expect(wrapper.find('[data-testid="trace-issue-summary"]').exists()).toBe(true)
  })

  it('shows empty state when no runs or traces', async () => {
    getAgentRuns.mockResolvedValue({ list: [] })
    getToolTraces.mockResolvedValue({ list: [] })

    const wrapper = await openPanel(3)

    expect(wrapper.find('[data-testid="trace-overview"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('本次会话暂无 Agent 执行记录')
  })

  it('shows live status for active run without canceling', async () => {
    getAgentRuns.mockResolvedValue({ list: [makeRun({ status: 'succeeded', error_message: '' })] })
    getToolTraces.mockResolvedValue({ list: [] })
    getActiveAgentRun.mockResolvedValue({
      has_active_run: true,
      run: {
        run_id: 77,
        session_id: 5,
        status: 'running',
        process_text: 'tool running',
        last_event_seq: 3,
      },
    })
    subscribeAgentRunEvents.mockImplementation(async (_id, _seq, handlers) => {
      handlers?.onEvent?.({
        run_id: 77,
        seq: 4,
        event_type: 'tool.started',
        status: 'running',
        tool_name: 'search_candidates',
      })
    })

    const wrapper = await openPanel(5)
    expect(wrapper.find('[data-testid="trace-live-status"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('实时执行中')
    expect(getActiveAgentRun).toHaveBeenCalled()
    expect(subscribeAgentRunEvents).toHaveBeenCalled()
    // panel must not import-call cancel
    const agentRun = await import('@/api/agentRun')
    expect(agentRun.cancelAgentRun).not.toHaveBeenCalled()
  })

  it('keeps historical data when subscription errors', async () => {
    getAgentRuns.mockResolvedValue({ list: [makeRun()] })
    getToolTraces.mockResolvedValue({ list: [] })
    getActiveAgentRun.mockResolvedValue({
      has_active_run: true,
      run: {
        run_id: 88,
        session_id: 6,
        status: 'running',
        last_event_seq: 0,
      },
    })
    subscribeAgentRunEvents.mockImplementation(async (_id, _seq, handlers) => {
      handlers?.onError?.({ message: 'sse down' })
    })

    const wrapper = await openPanel(6)
    expect(wrapper.text()).toContain('search_candidates')
    expect(wrapper.text()).toContain('sse down')
  })

  it('reconnects live trace from latest seq after an early stream close', async () => {
    vi.useFakeTimers()
    getAgentRuns.mockResolvedValue({ list: [makeRun({ status: 'succeeded', error_message: '' })] })
    getToolTraces.mockResolvedValue({ list: [] })
    getActiveAgentRun.mockResolvedValue({
      has_active_run: true,
      run: {
        run_id: 91,
        session_id: 8,
        status: 'running',
        process_text: 'tool running',
        last_event_seq: 3,
      },
    })
    subscribeAgentRunEvents.mockImplementation(async (_id, _seq, handlers) => {
      if (subscribeAgentRunEvents.mock.calls.length === 1) {
        handlers?.onEvent?.({
          run_id: 91,
          seq: 4,
          event_type: 'process.delta',
          status: 'running',
          delta: 'still running',
        })
        handlers?.onDone?.()
      }
    })

    await openPanel(8)
    expect(subscribeAgentRunEvents).toHaveBeenCalledTimes(1)

    await vi.advanceTimersByTimeAsync(400)
    await Promise.resolve()

    expect(subscribeAgentRunEvents).toHaveBeenCalledTimes(2)
    expect(subscribeAgentRunEvents.mock.calls[1][0]).toBe(91)
    expect(subscribeAgentRunEvents.mock.calls[1][1]).toBe(4)
  })

  it('shows frontend long-history truncation without backend pagination', async () => {
    const manyRuns = Array.from({ length: 25 }, (_, i) => makeRun({
      id: i + 1,
      status: 'succeeded',
      error_message: '',
      steps: [],
    }))
    getAgentRuns.mockResolvedValue({ list: manyRuns })
    getToolTraces.mockResolvedValue({ list: [] })

    const wrapper = await openPanel(7)
    expect(wrapper.find('[data-testid="trace-history-truncation"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="trace-load-more-runs"]').exists()).toBe(true)
    await wrapper.get('[data-testid="trace-load-more-runs"]').trigger('click')
    await nextTick()
    // still frontend-only; no extra list API for page 2
    expect(getAgentRuns).toHaveBeenCalledTimes(1)
  })

  it('supports filters and distinct filter-empty state', async () => {
    getAgentRuns.mockResolvedValue({ list: [makeRun()] })
    getToolTraces.mockResolvedValue({ list: [makeTrace()] })

    const wrapper = await openPanel(4)
    expect(wrapper.find('[data-testid="trace-filter-bar"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="trace-layers"]').exists()).toBe(true)

    const vm = wrapper.vm as unknown as {
      filterState: { keyword: string; statusFilter: string; typeFilter: string; issueOnly: boolean }
      isFilterEmpty: boolean
    }
    vm.filterState.keyword = 'definitely-missing-xyz'
    await nextTick()
    expect(wrapper.find('[data-testid="trace-filter-empty"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('没有符合当前筛选条件的轨迹')

    vm.filterState.keyword = ''
    vm.filterState.issueOnly = true
    await nextTick()
    expect(wrapper.find('[data-testid="trace-filter-empty"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('search_candidates')
  })
})
