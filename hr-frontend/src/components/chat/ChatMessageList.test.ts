import { describe, expect, it } from 'vitest'
import { shallowMount } from '@vue/test-utils'
import ChatMessageList from './ChatMessageList.vue'

const mountList = (retryDisabled: boolean) => shallowMount(ChatMessageList, {
  props: {
    messages: [{ role: 'assistant', content: '请求失败', failed: true, retryDisabled }],
    loading: false,
    streaming: false,
    sessionLoading: false,
    hasSession: true,
    runningModeLabel: 'Agent',
    renderMarkdown: (text: string) => text,
    waitingText: () => '响应中',
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
})
