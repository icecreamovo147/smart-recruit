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
