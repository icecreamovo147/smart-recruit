export type CapabilityStatus = 'available' | 'unavailable' | 'preparing' | 'enabled' | 'disabled' | 'active' | 'inactive' | 'draft' | string

export interface CapabilityPoint {
  id?: number
  name?: string
  display_name?: string
  description?: string
}

export interface CapabilityInfo {
  id: number
  skill_id?: number
  name: string
  display_name?: string
  category?: string
  description?: string
  scenarios?: string[] | string
  status?: CapabilityStatus
  tools_count?: number
  tools?: CapabilityPoint[]
  updated_at?: string
}

export interface CapabilityListResponse {
  list: CapabilityInfo[]
  total?: number
}

export interface CapabilityDetailResponse {
  capability: CapabilityInfo
}

export interface CreateCapabilityFromTemplatePayload {
  template_key: string
  display_name: string
  description: string
  scenarios?: string[]
  instruction?: string
}
