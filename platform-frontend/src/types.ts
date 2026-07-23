export interface PlatformUser {
  user_id: number
  username: string
  email?: string
  account_type: string
  roles: string[]
  permissions: string[]
  client_app: string
  available_apps: string[]
}

export interface PlatformAccount {
  user_id: number
  username: string
  email: string
  status: 'active' | 'disabled'
  roles: string[]
  token_version: number
  created_at: string
}

export interface Tenant {
  id: number
  tenant_key: string
  slug: string
  name: string
  status: 'active' | 'suspended' | 'disabled'
  timezone: string
  locale: string
  is_default: boolean
  membership_count: number
  created_at: string
  updated_at: string
}

export interface Membership {
  membership_id: number
  tenant_id: number
  user_id: number
  username: string
  email: string
  tenant_key: string
  slug: string
  name: string
  tenant_status: string
  membership_status: 'active' | 'suspended'
  roles: string[]
  data_scopes?: string[]
  joined_at: string
}

export interface PlatformDashboard {
  total_tenants: number
  active_tenants: number
  suspended_tenants: number
  disabled_tenants: number
  new_tenants_30d: number
  total_memberships: number
  active_memberships: number
  tenants_without_admin: number
  status_distribution: Array<{ status: string; count: number }>
}

export interface PlatformAuditLog {
  id: number
  actor_user_id: number
  actor_username: string
  action: string
  resource_type: string
  resource_id: number
  target_tenant_id: number
  target_tenant_name: string
  before_json: string
  after_json: string
  request_id: string
  client_ip: string
  created_at: string
}

export interface PageResult<T> {
  total: number
  list: T[]
}

export interface PlatformEntitlement {
  key: string
  value_type: 'integer' | 'boolean' | 'string'
  value_json: string
  enforcement_mode: 'hard' | 'soft' | 'observe'
  source?: 'plan' | 'override'
}

export interface PlatformPlanVersion {
  id: number
  plan_id: number
  version: number
  status: 'draft' | 'published' | 'retired'
  effective_at: string
  retired_at: string
  change_note: string
  entitlements: PlatformEntitlement[]
  created_at: string
  updated_at: string
}

export interface PlatformPlan {
  id: number
  plan_key: string
  name: string
  description: string
  status: 'active' | 'retired'
  versions: PlatformPlanVersion[]
  created_at: string
  updated_at: string
}

export interface TenantSubscription {
  id: number
  tenant_id: number
  plan_version_id: number
  plan_key: string
  plan_name: string
  plan_version: number
  status: 'scheduled' | 'active' | 'expired' | 'cancelled'
  starts_at: string
  ends_at: string
  reason: string
  effective_entitlements: PlatformEntitlement[]
}

export interface TenantUsageMetric {
  key: string
  usage_value: number
  quota_value: number
  usage_percent: number
  enforcement_mode: string
  measured_at: string
}

export interface QuotaAlert {
  id: number
  tenant_id: number
  tenant_name: string
  metric_key: string
  threshold_percent: 80 | 90 | 100
  usage_value: number
  quota_value: number
  status: 'open' | 'acknowledged' | 'resolved'
  assignee_user_id: number
  acknowledged_at: string
  resolved_at: string
  resolution_note: string
  first_triggered_at: string
  last_triggered_at: string
}
