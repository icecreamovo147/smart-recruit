export interface PlatformUser {
  user_id: number
  username: string
  account_type: string
  roles: string[]
  permissions: string[]
  client_app: string
  available_apps: string[]
}

export interface Tenant {
  id: number
  tenant_key: string
  slug: string
  name: string
  status: string
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
  tenant_key: string
  slug: string
  name: string
  tenant_status: string
  membership_status: string
  roles: string[]
  data_scopes: string[]
}
