import type { BillingOrder } from '@shared/types/billing'

export const isBillingOrderPayable = (order: BillingOrder, nowMs: number): boolean => (
  (order.status === 'pending' || order.status === 'paying')
  && Number(order.expires_at_unix_ms) > nowMs
)

export const effectiveBillingOrderStatus = (order: BillingOrder, nowMs: number): string => {
  if (isBillingOrderPayable(order, nowMs)) return order.status
  return order.status === 'pending' || order.status === 'paying' ? 'closed' : order.status
}

export const billingOrderRemainingLabel = (order: BillingOrder, nowMs: number): string => {
  const remainingSeconds = Math.max(0, Math.ceil((Number(order.expires_at_unix_ms) - nowMs) / 1000))
  if (remainingSeconds <= 0) return '已超时'
  const minutes = Math.floor(remainingSeconds / 60)
  const seconds = remainingSeconds % 60
  return `${minutes}分${String(seconds).padStart(2, '0')}秒`
}
