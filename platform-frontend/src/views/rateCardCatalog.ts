import type { LlmModel, LlmProvider } from '@shared/types/llm'

export const enabledRateProviders = (providers: LlmProvider[]): LlmProvider[] => providers
  .filter((provider) => provider.is_enabled)
  .sort((left, right) => left.name.localeCompare(right.name))

export const enabledRateModelsForProvider = (
  providers: LlmProvider[],
  models: LlmModel[],
  providerKey: string,
): LlmModel[] => {
  const provider = providers.find((item) => item.is_enabled && item.name === providerKey)
  if (!provider) return []
  return models
    .filter((model) => model.is_enabled && model.provider_id === provider.id)
    .sort((left, right) => (left.display_name || left.model_name).localeCompare(right.display_name || right.model_name))
}

export const isEnabledRateTarget = (
  providers: LlmProvider[],
  models: LlmModel[],
  providerKey: string,
  modelKey: string,
): boolean => enabledRateModelsForProvider(providers, models, providerKey)
  .some((model) => model.model_name === modelKey)
