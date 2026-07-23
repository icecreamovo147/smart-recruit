export interface LlmModel {
  id: number
  model_name: string
  display_name: string
  is_enabled: boolean
  is_default: boolean
  max_tokens?: number
  context_window_tokens?: number
}

export interface PaginatedList<T> {
  total: number
  list: T[]
}
