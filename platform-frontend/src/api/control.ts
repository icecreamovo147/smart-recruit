import http from './http'
import type { PageResult, PlatformEntitlement, PlatformPlan, PlatformPlanVersion, QuotaAlert, TenantSubscription, TenantUsageMetric } from '@/types'

export interface BillingPriceAdmin { id: number; version: number; billing_term: 'monthly' | 'yearly' | 'one_time'; amount_fen: number; included_credits: number; entitlement_snapshot_json: string }
export interface BillingProductAdmin { id: number; product_key: string; name: string; description: string; product_type: 'subscription' | 'credit_pack'; prices: BillingPriceAdmin[] }
export interface AIRateCardAdmin { id: number; provider_key: string; model_key: string; version: number; currency: string; input_micros_per_1k_tokens: number; output_micros_per_1k_tokens: number; cached_input_micros_per_1k_tokens: number; credit_micros: number; status: string; effective_at_unix_ms: number }

export const listPlans = (status = '') => http.get<never, { list: PlatformPlan[] }>('/api/v1/platform/plans', { params: { status } })

export const savePlanVersion = (planId: number, payload: { version_id?: number; change_note: string; entitlements: PlatformEntitlement[] }) =>
  http.post<never, { version: PlatformPlanVersion }>(`/api/v1/platform/plans/${planId}/versions`, payload)

export const publishPlanVersion = (planId: number, versionId: number, payload: { effective_at: string; reason: string }) =>
  http.post<never, { version: PlatformPlanVersion }>(`/api/v1/platform/plans/${planId}/versions/${versionId}/publish`, payload)

export const listBillingProducts = () => http.get<never, { products: BillingProductAdmin[]; payment_environment: string }>('/api/v1/platform/billing/products')
export const saveBillingPrice = (payload: { product_id: number; price_version_id?: number; billing_term: string; amount_fen: number; included_credits: number; entitlement_snapshot_json: string; publish: boolean }) => http.post('/api/v1/platform/billing/prices', payload)
export const listAIRateCards = () => http.get<never, { rates: AIRateCardAdmin[] }>('/api/v1/platform/billing/rates')
export const saveAIRateCard = (payload: Omit<AIRateCardAdmin, 'id' | 'version' | 'currency' | 'status' | 'effective_at_unix_ms'> & { publish: boolean }) => http.post('/api/v1/platform/billing/rates', payload)

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
