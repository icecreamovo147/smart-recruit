import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { nextTick, reactive } from 'vue'
import { describe, expect, it } from 'vitest'
import AgentSkillPackageEditor from './AgentSkillPackageEditor.vue'
import { createEmptyPackageEditorModel } from './packageEditor'

describe('AgentSkillPackageEditor', () => {
  it('adds an ordered reference section and displays derived activation policy', async () => {
    const model = reactive(createEmptyPackageEditorModel())
    model.risk = 'critical'
    const wrapper = mount(AgentSkillPackageEditor, {
      props: { modelValue: model },
      global: { plugins: [ElementPlus] },
    })

    await nextTick()
    const activationInput = wrapper.get('[data-testid="activation-policy"]')
    expect((activationInput.element as HTMLInputElement).value).toBe('manual_only')

    await wrapper.get('[data-testid="add-section"]').trigger('click')
    expect(model.sections).toHaveLength(1)
    expect(model.sections[0]).toMatchObject({ sectionKey: 'reference_1', title: '参考资料 1' })
  })

  it('forces supporting skills to use no output contract', async () => {
    const model = reactive(createEmptyPackageEditorModel())
    model.outputMode = 'advisory'
    const wrapper = mount(AgentSkillPackageEditor, {
      props: { modelValue: model },
      global: { plugins: [ElementPlus] },
    })

    model.compositionRole = 'supporting'
    await nextTick()
    expect(model.outputMode).toBe('none')
    expect(model.outputSchemaId).toBe('')
    expect(model.outputSchemaJson).toBe('')
    wrapper.unmount()
  })

  it('reuses the first available generated key after deleting a section', async () => {
    const model = reactive(createEmptyPackageEditorModel())
    const wrapper = mount(AgentSkillPackageEditor, {
      props: { modelValue: model },
      global: { plugins: [ElementPlus] },
    })

    await wrapper.get('[data-testid="add-section"]').trigger('click')
    await wrapper.get('[data-testid="add-section"]').trigger('click')
    expect(model.sections.map((section) => section.sectionKey)).toEqual([
      'reference_1',
      'reference_2',
    ])

    await wrapper.get('[data-testid="delete-section-0"]').trigger('click')
    await wrapper.get('[data-testid="add-section"]').trigger('click')
    expect(model.sections.map((section) => section.sectionKey)).toEqual([
      'reference_2',
      'reference_1',
    ])
    wrapper.unmount()
  })
})
