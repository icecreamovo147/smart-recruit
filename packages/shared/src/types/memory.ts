export const MEMORY_OWNER_ROLE_CANDIDATE = 1
export const MEMORY_OWNER_ROLE_HR = 2

export interface MemoryInfo {
  id: number
  tenant_id?: number
  owner_role: number
  owner_id: number
  hr_id?: number
  scope_type: string
  scope_id: number
  memory_type: string
  content: string
  source: string
  confidence: number
  importance: number
  status: string
  pii_level?: string
  content_hash?: string
  created_at?: string
  updated_at?: string
  revoked_by?: number
  revoke_reason?: string
}

export interface MemoryListParams {
  owner_role?: number
  owner_id?: number
  scope_type?: string
  scope_id?: number
  memory_type?: string
  status?: string
  page?: number
  page_size?: number
}

export interface MemoryListResult {
  total: number
  list: MemoryInfo[]
}

export interface CreateMemoryPayload {
  owner_role?: number
  owner_id?: number
  scope_type: string
  scope_id?: number
  memory_type: string
  content: string
  source?: string
  confidence?: number
  importance?: number
  /** Required when content is classified as high PII. */
  confirm_high_pii?: boolean
}

export interface RevokeMemoryPayload {
  owner_role?: number
  owner_id?: number
  revoke_reason?: string
}
