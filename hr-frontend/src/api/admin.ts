import request from './request'
import type { DepartmentLocationConfig, DepartmentNode, InviteCodeInfo, LocationOption, PaginatedList } from '@/types/domain'

export const createInviteCode = (expiresAt?: string): Promise<{
  invite_code: InviteCodeInfo
}> =>
  request.post('/api/v1/hr/admin/invite-codes', { expires_at: expiresAt || '' })

export const listInviteCodes = (page = 1, pageSize = 20): Promise<PaginatedList<InviteCodeInfo>> =>
  request.get('/api/v1/hr/admin/invite-codes', { params: { page, page_size: pageSize } })

export const extendInviteCode = (id: number, newExpiresAt: string): Promise<void> =>
  request.patch(`/api/v1/hr/admin/invite-codes/${id}/extend`, { new_expires_at: newExpiresAt })

export const revokeInviteCode = (id: number): Promise<void> =>
  request.patch(`/api/v1/hr/admin/invite-codes/${id}/revoke`)

export const reactivateInviteCode = (id: number): Promise<void> =>
  request.patch(`/api/v1/hr/admin/invite-codes/${id}/reactivate`)

// ── Department taxonomy ────────────────────────────────────────────

export const listDepartments = (): Promise<{ list: DepartmentNode[] }> =>
  request.get('/api/v1/hr/admin/departments')

export const createDepartment = (data: { parent_id: number; name: string; sort_order?: number }): Promise<{ department: DepartmentNode }> =>
  request.post('/api/v1/hr/admin/departments', data)

export const updateDepartment = (id: number, data: { parent_id?: number; name?: string; sort_order?: number }): Promise<{ department: DepartmentNode }> =>
  request.put(`/api/v1/hr/admin/departments/${id}`, data)

export const updateDepartmentStatus = (id: number, isActive: number): Promise<void> =>
  request.patch(`/api/v1/hr/admin/departments/${id}/status`, { is_active: isActive })

export const deleteDepartment = (id: number): Promise<void> =>
  request.delete(`/api/v1/hr/admin/departments/${id}`)

// ── Location taxonomy ──────────────────────────────────────────────

export const listLocations = (): Promise<{ list: LocationOption[] }> =>
  request.get('/api/v1/hr/admin/locations')

export const createLocation = (data: { name: string; code?: string; sort_order?: number }): Promise<{ location: LocationOption }> =>
  request.post('/api/v1/hr/admin/locations', data)

export const updateLocation = (id: number, data: { name?: string; code?: string; sort_order?: number }): Promise<{ location: LocationOption }> =>
  request.put(`/api/v1/hr/admin/locations/${id}`, data)

export const updateLocationStatus = (id: number, isActive: number): Promise<void> =>
  request.patch(`/api/v1/hr/admin/locations/${id}/status`, { is_active: isActive })

export const deleteLocation = (id: number): Promise<void> =>
  request.delete(`/api/v1/hr/admin/locations/${id}`)

// ── Department Location Config ─────────────────────────────────────

export const listDepartmentsLocationMap = (): Promise<{ items: { department_id: number; location_ids: number[] }[] }> =>
  request.get('/api/v1/hr/admin/departments/location-map')

export const getDepartmentLocationConfig = (departmentId: number): Promise<DepartmentLocationConfig> =>
  request.get(`/api/v1/hr/admin/departments/${departmentId}/locations`)

export const updateDepartmentLocationConfig = (departmentId: number, data: { inherit_locations: number; location_ids: number[] }): Promise<DepartmentLocationConfig> =>
  request.put(`/api/v1/hr/admin/departments/${departmentId}/locations`, data)

// ── Usage Audit ──────────────────────────────────────────────────────

export interface UsageLogQuery {
  page: number
  page_size: number
  service_type?: string
  provider?: string
  status?: string
  user_id?: number
  request_id?: string
  start_time?: string
  end_time?: string
}

export interface UsageLogItem {
  id: number
  user_id: number
  role: number
  service_type: string
  endpoint: string
  provider: string
  model: string
  request_chars: number
  response_chars: number
  estimated_tokens: number
  object_key: string
  object_size: number
  status: string
  error_code: string
  cost_ms: number
  request_id: string
  ip: string
  created_at: string
}

export const listUsageLogs = (params: UsageLogQuery): Promise<PaginatedList<UsageLogItem>> =>
  request.get('/api/v1/hr/admin/third-party-usage-logs', { params })
