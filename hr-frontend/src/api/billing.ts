import request from './request'
import type { BillingAccount, BillingOrder, BillingProduct } from '@shared/types/billing'

export type { BillingAccount, BillingOrder, BillingPrice, BillingProduct } from '@shared/types/billing'

export const getBillingCatalog = (): Promise<{ products: BillingProduct[]; payment_environment: string }> => request.get('/api/v1/hr/billing/catalog')
export const getBillingAccount = (): Promise<BillingAccount> => request.get('/api/v1/hr/billing/subscription')
export const listBillingOrders = (): Promise<{ orders: BillingOrder[]; total: number }> => request.get('/api/v1/hr/billing/orders?page_size=100')
export const createBillingOrder = (priceVersionId: number, orderType: string, replacePendingOrder = false): Promise<{ order: BillingOrder }> => request.post('/api/v1/hr/billing/orders', {
  price_version_id: priceVersionId,
  order_type: orderType,
  idempotency_key: crypto.randomUUID(),
  replace_pending_order: replacePendingOrder,
})
export const payBillingOrder = (
  orderNo: string,
  scene: 'desktop' | 'wap' | 'sync',
  options?: { silentError?: boolean },
): Promise<{ redirect_url: string; payment_environment: string; payment_no?: string }> => (
  request.post(`/api/v1/hr/billing/orders/${orderNo}/pay`, { scene }, { silentError: options?.silentError })
)
export const refundBillingOrder = (orderNo: string, reason: string) => request.post(`/api/v1/hr/billing/orders/${orderNo}/refund`, { reason })
