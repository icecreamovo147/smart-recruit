import http from './http'
import type { Membership, PageResult, Tenant } from '@/types'

export const listTenants = (params: { page: number; page_size: number; keyword?: string; status?: string }): Promise<PageResult<Tenant>> =>
  http.get('/api/v1/platform/tenants', { params })

export const createTenant = (payload: { slug: string; name: string; timezone: string; locale: string }): Promise<{ tenant: Tenant }> =>
  http.post('/api/v1/platform/tenants', payload)

export const getTenant = (tenantId: number): Promise<{ tenant: Tenant }> =>
  http.get(`/api/v1/platform/tenants/${tenantId}`)

export const updateTenantStatus = (tenantId: number, status: string, reason: string): Promise<{ tenant: Tenant }> =>
  http.patch(`/api/v1/platform/tenants/${tenantId}/status`, { status, reason })

export const listMemberships = (tenantId: number, params: { page: number; page_size: number }): Promise<PageResult<Membership>> =>
  http.get(`/api/v1/platform/tenants/${tenantId}/memberships`, { params })

export const updateMembershipStatus = (tenantId: number, membershipId: number, status: string, reason: string): Promise<{ membership: Membership }> =>
  http.patch(`/api/v1/platform/tenants/${tenantId}/memberships/${membershipId}/status`, { status, reason })
