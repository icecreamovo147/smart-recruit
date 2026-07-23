import { describe, expect, it } from 'vitest'
import type { BillingOrder } from '@shared/types/billing'
import { billingOrderRemainingLabel, effectiveBillingOrderStatus, isBillingOrderPayable } from './billingOrder'

const order = (status: string, expiresAt: number): BillingOrder => ({
  order_no: 'B20260720001',
  order_type: 'subscribe',
  product_key: 'tenant_basic',
  product_name: '企业基础版',
  amount_fen: 3900,
  status,
  payment_environment: 'sandbox',
  expires_at_unix_ms: expiresAt,
  created_at_unix_ms: expiresAt - 30 * 60 * 1000,
})

describe('billing order payment window', () => {
  it('keeps a pending order payable before its fixed deadline', () => {
    const current = order('pending', 100_000)
    expect(isBillingOrderPayable(current, 40_000)).toBe(true)
    expect(effectiveBillingOrderStatus(current, 40_000)).toBe('pending')
    expect(billingOrderRemainingLabel(current, 40_000)).toBe('1分00秒')
  })

  it('projects an unpaid order as closed at the deadline', () => {
    const expired = order('paying', 100_000)
    expect(isBillingOrderPayable(expired, 100_000)).toBe(false)
    expect(effectiveBillingOrderStatus(expired, 100_000)).toBe('closed')
    expect(billingOrderRemainingLabel(expired, 100_000)).toBe('已超时')
  })

  it('does not rewrite terminal order states', () => {
    expect(effectiveBillingOrderStatus(order('paid', 10_000), 100_000)).toBe('paid')
  })
})
