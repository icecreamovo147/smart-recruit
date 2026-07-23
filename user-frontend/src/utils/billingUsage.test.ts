import { describe, expect, it } from 'vitest'
import type { BillingAccount } from '@shared/types/billing'
import { calculateBillingUsagePercent } from './billingUsage'

const account = (overrides: Partial<BillingAccount>): BillingAccount => ({
  available_credits: 0,
  reserved_credits: 0,
  next_expiry_at_unix_ms: 0,
  payment_environment: 'sandbox',
  total_credits: 0,
  used_credits: 0,
  next_refresh_at_unix_ms: 0,
  ...overrides,
})

describe('calculateBillingUsagePercent', () => {
  it('calculates a normal consumed percentage', () => {
    expect(calculateBillingUsagePercent(account({ total_credits: 100, used_credits: 20, available_credits: 80 }))).toBe(20)
  })

  it('combines consumed monthly credits with a newly purchased credit pack', () => {
    expect(calculateBillingUsagePercent(account({ total_credits: 200, used_credits: 100, available_credits: 100 }))).toBe(50)
  })

  it('renders an exhausted subscription as 100% even for legacy zero-string payloads', () => {
    expect(calculateBillingUsagePercent(account({
      total_credits: '0' as unknown as number,
      used_credits: '0' as unknown as number,
      available_credits: '0' as unknown as number,
      subscription: {
        id: 1,
        product_key: 'candidate_free',
        product_name: '求职者免费版',
        status: 'active',
        term: 'monthly',
        current_period_start_unix_ms: 0,
        current_period_end_unix_ms: 0,
        cancel_at_period_end: false,
        source: 'free_tier',
        included_credits: 100,
      },
    }))).toBe(100)
  })

  it('never returns NaN for invalid payloads', () => {
    expect(calculateBillingUsagePercent(account({ total_credits: Number.NaN, used_credits: Number.NaN }))).toBe(0)
  })
})
