import { describe, expect, it } from 'vitest'
import type { LlmModel, LlmProvider } from '@shared/types/llm'
import { enabledRateModelsForProvider, enabledRateProviders, isEnabledRateTarget } from './rateCardCatalog'

const providers = [
  { id: 1, name: 'DeepSeek', is_enabled: true },
  { id: 2, name: 'Disabled', is_enabled: false },
] as LlmProvider[]

const models = [
  { id: 1, provider_id: 1, model_name: 'deepseek-chat', display_name: 'DeepSeek Chat', is_enabled: true },
  { id: 2, provider_id: 1, model_name: 'deepseek-old', display_name: 'Old', is_enabled: false },
  { id: 3, provider_id: 2, model_name: 'disabled-model', display_name: 'Disabled Model', is_enabled: true },
] as LlmModel[]

describe('rate card model catalog', () => {
  it('only exposes enabled providers and enabled models belonging to the selected provider', () => {
    expect(enabledRateProviders(providers).map((item) => item.name)).toEqual(['DeepSeek'])
    expect(enabledRateModelsForProvider(providers, models, 'DeepSeek').map((item) => item.model_name)).toEqual(['deepseek-chat'])
  })

  it('requires an exact maintained provider/model pair', () => {
    expect(isEnabledRateTarget(providers, models, 'DeepSeek', 'deepseek-chat')).toBe(true)
    expect(isEnabledRateTarget(providers, models, 'deepseek', 'deepseek-chat')).toBe(false)
    expect(isEnabledRateTarget(providers, models, 'DeepSeek', 'deepseek-old')).toBe(false)
    expect(isEnabledRateTarget(providers, models, 'Disabled', 'disabled-model')).toBe(false)
  })
})
