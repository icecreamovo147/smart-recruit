import http from './http'
import type { PageResult, PlatformEntitlement, PlatformPlan, PlatformPlanVersion, QuotaAlert, TenantSubscription, TenantUsageMetric } from '@/types'

export const listPlans = (status = '') => http.get<never, { list: PlatformPlan[] }>('/api/v1/platform/plans', { params: { status } })

export const savePlanVersion = (planId: number, payload: { version_id?: number; change_note: string; entitlements: PlatformEntitlement[] }) =>
  http.post<never, { version: PlatformPlanVersion }>(`/api/v1/platform/plans/${planId}/versions`, payload)

export const publishPlanVersion = (planId: number, versionId: number, payload: { effective_at: string; reason: string }) =>
  http.post<never, { version: PlatformPlanVersion }>(`/api/v1/platform/plans/${planId}/versions/${versionId}/publish`, payload)

export const getTenantSubscription = (tenantId: number) =>
  http.get<never, { subscription?: TenantSubscription }>(`/api/v1/platform/tenants/${tenantId}/subscription`)

export const updateTenantSubscription = (tenantId: number, payload: { plan_version_id: number; starts_at: string; ends_at?: string; reason: string }) =>
  http.put<never, { subscription: TenantSubscription }>(`/api/v1/platform/tenants/${tenantId}/subscription`, payload)

export const updateTenantEntitlementOverride = (
  tenantId: number,
  payload: { entitlement_key: string; value_type: 'integer'; value_json: string; expires_at?: string; reason: string },
) => http.put<never, { subscription: TenantSubscription }>(`/api/v1/platform/tenants/${tenantId}/entitlement-override`, payload)

export const getTenantUsage = (tenantId: number) =>
  http.get<never, { tenant_id: number; metrics: TenantUsageMetric[] }>(`/api/v1/platform/tenants/${tenantId}/usage`)

export const listQuotaAlerts = (params: Record<string, string | number | undefined>) =>
  http.get<never, PageResult<QuotaAlert>>('/api/v1/platform/quota-alerts', { params })

export const updateQuotaAlert = (alertId: number, payload: { status: 'acknowledged' | 'resolved'; assignee_user_id?: number; resolution_note?: string }) =>
  http.patch<never, { alert: QuotaAlert }>(`/api/v1/platform/quota-alerts/${alertId}`, payload)
