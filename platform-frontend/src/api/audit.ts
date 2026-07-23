import http from './http'
import type { PageResult, PlatformAuditLog } from '@/types'

export interface AuditQuery {
  page: number
  page_size: number
  tenant_id?: number
  actor_user_id?: number
  action?: string
  request_id?: string
  start_time?: string
  end_time?: string
}

export const listPlatformAuditLogs = (params: AuditQuery): Promise<PageResult<PlatformAuditLog>> =>
  http.get('/api/v1/platform/audit-logs', { params })
