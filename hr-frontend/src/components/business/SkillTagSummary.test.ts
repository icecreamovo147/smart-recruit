import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SkillTagSummary from './SkillTagSummary.vue'

const mountSummary = (skills: string[], limit = 3) => mount(SkillTagSummary, {
  props: { skills, limit },
  global: {
    stubs: {
      ElTag: { template: '<span class="el-tag-stub"><slot /></span>' },
      ElPopover: {
        name: 'ElPopover',
        props: ['trigger'],
        template: '<div class="el-popover-stub"><slot name="reference" /><slot /></div>',
      },
    },
  },
})

describe('SkillTagSummary', () => {
  it('shows a limited summary and the remaining count', () => {
    const wrapper = mountSummary(['Java', 'Spring Boot', 'MySQL', 'Redis', 'Docker'])

    const summaryTags = wrapper.findAll('.skill-tag-summary > .el-tag-stub')
    expect(summaryTags.map((tag) => tag.text())).toEqual(['Java', 'Spring Boot', 'MySQL'])
    expect(wrapper.get('.skill-tag-summary__more').text()).toBe('+2')
    expect(wrapper.get('.skill-tag-summary__popover').text()).toContain('Docker')
    expect(wrapper.getComponent({ name: 'ElPopover' }).props('trigger')).toBe('hover')
  })

  it('trims and deduplicates skills without changing their order', () => {
    const wrapper = mountSummary([' Java ', 'java', '', 'Vue', ' VUE ', 'Go'], 2)

    const summaryTags = wrapper.findAll('.skill-tag-summary > .el-tag-stub')
    expect(summaryTags.map((tag) => tag.text())).toEqual(['Java', 'Vue'])
    expect(wrapper.get('.skill-tag-summary__more').text()).toBe('+1')
  })

  it('renders an empty placeholder when no skills are available', () => {
    const wrapper = mountSummary([])

    expect(wrapper.get('.skill-tag-summary__empty').text()).toBe('-')
    expect(wrapper.find('.skill-tag-summary__more').exists()).toBe(false)
  })
})
