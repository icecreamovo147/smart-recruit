export type BillingTerm = 'monthly' | 'yearly' | 'one_time'
export type BillingProductType = 'subscription' | 'credit_pack'
export type BillingPlanSource = 'paid_subscription' | 'platform_plan' | 'free_tier'

export interface BillingPrice {
  id: number
  version: number
  billing_term: BillingTerm
  amount_fen: number
  currency: string
  included_credits: number
}

export interface BillingProduct {
  id: number
  product_key: string
  name: string
  description: string
  product_type: BillingProductType
  prices: BillingPrice[]
}

export interface BillingSubscription {
  id: number
  product_key: string
  product_name: string
  status: string
  term: BillingTerm
  current_period_start_unix_ms: number
  current_period_end_unix_ms: number
  cancel_at_period_end: boolean
  source: BillingPlanSource
  included_credits: number
}

export interface BillingAccount {
  subscription?: BillingSubscription
  available_credits: number
  reserved_credits: number
  next_expiry_at_unix_ms: number
  payment_environment: string
  total_credits: number
  used_credits: number
  next_refresh_at_unix_ms: number
}

export interface BillingOrder {
  order_no: string
  order_type: string
  product_key: string
  product_name: string
  amount_fen: number
  status: string
  payment_environment: string
  expires_at_unix_ms: number
  created_at_unix_ms: number
}
