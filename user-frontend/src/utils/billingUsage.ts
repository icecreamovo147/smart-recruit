import type { BillingAccount } from '@shared/types/billing'

export const calculateBillingUsagePercent = (account: BillingAccount | null): number => {
  if (!account) return 0
  const total = Number(account.total_credits)
  const used = Number(account.used_credits)
  if (!Number.isFinite(total) || total <= 0) {
    return Number(account.available_credits) <= 0 && account.subscription ? 100 : 0
  }
  if (!Number.isFinite(used) || used <= 0) return 0
  return Math.min(100, Math.max(0, Math.round((used / total) * 100)))
}
