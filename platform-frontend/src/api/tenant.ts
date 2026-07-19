import http from './http'
import type { Membership, Tenant } from '@/types'

export const listTenants = (params: { page: number; page_size: number; keyword?: string; status?: string }): Promise<{ total: number; list: Tenant[] }> =>
  http.get('/api/v1/platform/tenants', { params })

export const createTenant = (payload: { slug: string; name: string; timezone: string; locale: string }): Promise<{ tenant: Tenant }> =>
  http.post('/api/v1/platform/tenants', payload)

export const updateTenantStatus = (tenantId: number, status: string): Promise<{ tenant: Tenant }> =>
  http.patch(`/api/v1/platform/tenants/${tenantId}/status`, { status })

export const listMemberships = (tenantId: number): Promise<{ total: number; list: Membership[] }> =>
  http.get(`/api/v1/platform/tenants/${tenantId}/memberships`, { params: { page: 1, page_size: 100 } })
