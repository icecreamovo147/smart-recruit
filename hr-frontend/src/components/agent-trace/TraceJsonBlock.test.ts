import { describe, expect, it, vi, beforeEach } from 'vitest'
import { defineComponent, h, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import TraceJsonBlock from './TraceJsonBlock.vue'

const message = vi.hoisted(() => ({
  success: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
}))

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<typeof import('element-plus')>('element-plus')
  return {
    ...actual,
    ElMessage: message,
  }
})

function mountBlock(content: string) {
  return mount(TraceJsonBlock, {
    props: { content, label: '输入' },
    global: {
      stubs: {
        'el-tag': true,
        'el-button': defineComponent({
          name: 'el-button',
          emits: ['click'],
          setup(_, { slots, emit }) {
            return () => h('button', { onClick: () => emit('click') }, slots.default?.())
          },
        }),
      },
    },
  })
}

describe('TraceJsonBlock', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    })
  })

  it('pretty-formats valid JSON and collapses long content', async () => {
    const long = JSON.stringify({ a: 1, b: 'x'.repeat(300) })
    const wrapper = mountBlock(long)
    expect(wrapper.get('[data-testid="trace-json-content"]').text()).toContain('"a": 1')
    expect(wrapper.find('[data-testid="trace-json-toggle"]').exists()).toBe(true)
    await wrapper.get('[data-testid="trace-json-toggle"]').trigger('click')
    await nextTick()
    expect(wrapper.get('[data-testid="trace-json-content"]').text().length).toBeGreaterThan(240)
  })

  it('expands nested JSON-encoded result strings in the viewer', () => {
    const content = JSON.stringify({
      result: JSON.stringify({ total: 3, jobs: [{ job_id: 1, title: '工程师' }] }),
      tool_call_id: 'call_x',
    })
    const wrapper = mountBlock(content)
    const text = wrapper.get('[data-testid="trace-json-content"]').text()
    expect(text).toContain('"total": 3')
    expect(text).toContain('"job_id": 1')
    expect(text).toContain('"tool_call_id": "call_x"')
    expect(text).not.toContain('\\"total\\"')
  })

  it('preserves invalid JSON as raw text', () => {
    const wrapper = mountBlock('not-json {')
    expect(wrapper.get('[data-testid="trace-json-content"]').text()).toBe('not-json {')
  })

  it('copies full desensitized content', async () => {
    const wrapper = mountBlock('{"phone":"***"}')
    await wrapper.get('[data-testid="trace-json-copy"]').trigger('click')
    await nextTick()
    expect(navigator.clipboard.writeText).toHaveBeenCalled()
    const arg = (navigator.clipboard.writeText as ReturnType<typeof vi.fn>).mock.calls[0][0] as string
    expect(arg).toContain('***')
    expect(message.success).toHaveBeenCalled()
  })

  it('handles copy failure without crashing', async () => {
    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn().mockRejectedValue(new Error('denied')),
      },
    })
    const wrapper = mountBlock('{"ok":true}')
    await wrapper.get('[data-testid="trace-json-copy"]').trigger('click')
    await nextTick()
    expect(message.warning).toHaveBeenCalled()
  })
})
