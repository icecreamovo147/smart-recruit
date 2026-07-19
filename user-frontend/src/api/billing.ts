import request from './request'

export interface BillingPrice { id: number; version: number; billing_term: 'monthly' | 'yearly' | 'one_time'; amount_fen: number; currency: string; included_credits: number }
export interface BillingProduct { id: number; product_key: string; name: string; description: string; product_type: 'subscription' | 'credit_pack'; prices: BillingPrice[] }
export interface BillingAccount { subscription?: { product_key: string; product_name: string; status: string; term: string; current_period_end_unix_ms: number }; available_credits: number; reserved_credits: number; next_expiry_at_unix_ms: number; payment_environment: string }
export interface BillingOrder { order_no: string; order_type: string; product_name: string; amount_fen: number; status: string; payment_environment: string; expires_at_unix_ms: number; created_at_unix_ms: number }

export const getBillingCatalog = (): Promise<{ products: BillingProduct[]; payment_environment: string }> => request.get('/api/v1/candidate/billing/catalog')
export const getBillingAccount = (): Promise<BillingAccount> => request.get('/api/v1/candidate/billing/subscription')
export const listBillingOrders = (): Promise<{ orders: BillingOrder[]; total: number }> => request.get('/api/v1/candidate/billing/orders')
export const createBillingOrder = (priceVersionId: number, orderType: string): Promise<{ order: BillingOrder }> => request.post('/api/v1/candidate/billing/orders', { price_version_id: priceVersionId, order_type: orderType, idempotency_key: crypto.randomUUID() })
export const payBillingOrder = (orderNo: string, scene: 'desktop' | 'wap'): Promise<{ redirect_url: string; payment_environment: string }> => request.post(`/api/v1/candidate/billing/orders/${orderNo}/pay`, { scene })
export const refundBillingOrder = (orderNo: string, reason: string) => request.post(`/api/v1/candidate/billing/orders/${orderNo}/refund`, { reason })
