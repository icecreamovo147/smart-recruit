import http from './http'
import type { PageResult, PlatformEntitlement, PlatformPlan, PlatformPlanVersion, QuotaAlert, TenantSubscription, TenantUsageMetric } from '@/types'

export interface BillingPriceAdmin { id: number; version: number; billing_term: 'monthly' | 'yearly' | 'one_time'; amount_fen: number; included_credits: number; entitlement_snapshot_json: string }
export interface BillingProductAdmin { id: number; product_key: string; name: string; description: string; product_type: 'subscription' | 'credit_pack'; prices: BillingPriceAdmin[] }
export interface AIRateCardAdmin { id: number; provider_key: string; model_key: string; version: number; currency: string; input_micros_per_1k_tokens: number; output_micros_per_1k_tokens: number; cached_input_micros_per_1k_tokens: number; credit_micros: number; status: string; effective_at_unix_ms: number }
export interface BillingRefundAdmin { refund_no: string; order_no: string; owner_type: string; owner_id: number; amount_fen: number; status: string; review_mode: string; reason: string; requested_by: number; reviewed_by: number; created_at_unix_ms: number; last_error: string }

const numericID = (value: unknown): number => {
  const normalized = Number(value)
  return Number.isSafeInteger(normalized) && normalized > 0 ? normalized : 0
}

const normalizePlanVersion = (version: PlatformPlanVersion): PlatformPlanVersion => ({
  ...version,
  id: numericID(version.id),
  plan_id: numericID(version.plan_id),
  version: Number(version.version) || 0,
})

const normalizePlan = (plan: PlatformPlan): PlatformPlan => ({
  ...plan,
  id: numericID(plan.id),
  versions: (plan.versions || []).map(normalizePlanVersion),
})

const normalizeSubscription = (subscription: TenantSubscription): TenantSubscription => ({
  ...subscription,
  id: numericID(subscription.id),
  tenant_id: numericID(subscription.tenant_id),
  plan_version_id: numericID(subscription.plan_version_id),
  plan_version: Number(subscription.plan_version) || 0,
})

export const listPlans = async (status = '') => {
  const result = await http.get<never, { list: PlatformPlan[] }>('/api/v1/platform/plans', { params: { status } })
  return { ...result, list: (result.list || []).map(normalizePlan) }
}

export const savePlanVersion = (planId: number, payload: { version_id?: number; change_note: string; entitlements: PlatformEntitlement[] }) =>
  http.post<never, { version: PlatformPlanVersion }>(`/api/v1/platform/plans/${planId}/versions`, payload)

export const publishPlanVersion = (planId: number, versionId: number, payload: { effective_at: string; reason: string }) =>
  http.post<never, { version: PlatformPlanVersion }>(`/api/v1/platform/plans/${planId}/versions/${versionId}/publish`, payload)

export const listBillingProducts = () => http.get<never, { products: BillingProductAdmin[]; payment_environment: string }>('/api/v1/platform/billing/products')
export const saveBillingPrice = (payload: { product_id: number; price_version_id?: number; billing_term: string; amount_fen: number; included_credits: number }) => http.post('/api/v1/platform/billing/prices', payload)
export const listAIRateCards = () => http.get<never, { rates: AIRateCardAdmin[] }>('/api/v1/platform/billing/rates')
export const saveAIRateCard = (payload: Omit<AIRateCardAdmin, 'id' | 'version' | 'currency' | 'status' | 'effective_at_unix_ms'>) => http.post('/api/v1/platform/billing/rates', payload)
export const listBillingRefunds = () => http.get<never, { refunds: BillingRefundAdmin[]; total: number }>('/api/v1/platform/billing/refunds?page_size=100')
export const reviewBillingRefund = (refundNo: string, action: 'approve' | 'reject', reason = '') => http.post(`/api/v1/platform/billing/refunds/${refundNo}/review`, { action, reason })

export const getTenantSubscription = async (tenantId: number) => {
  const result = await http.get<never, { subscription?: TenantSubscription }>(`/api/v1/platform/tenants/${tenantId}/subscription`)
  return {
    ...result,
    subscription: result.subscription ? normalizeSubscription(result.subscription) : undefined,
  }
}

export const updateTenantSubscription = async (tenantId: number, payload: { plan_version_id: number; starts_at: string; ends_at?: string; reason: string }) => {
  const result = await http.put<never, { subscription: TenantSubscription }>(`/api/v1/platform/tenants/${tenantId}/subscription`, {
    ...payload,
    plan_version_id: numericID(payload.plan_version_id),
  })
  return { ...result, subscription: normalizeSubscription(result.subscription) }
}

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
