import request from './request'
import type { PaginatedList } from '@/types/domain'
import type {
  McpServerInfo,
  McpToolInfo,
  McpToolPolicy,
  McpCallLog,
  CreateMcpServerPayload,
  UpdateMcpServerPayload,
  CreateMcpToolPolicyPayload,
  UpdateMcpToolPolicyPayload,
  McpLogQuery,
  McpPolicyQuery,
  McpTestResult,
} from '@/types/mcp'

// ── Response transformer ─────────────────────────────────────────────────

/** Transform a backend server object to frontend McpServerInfo shape */
function transformServer(raw: Record<string, unknown>): McpServerInfo {
  const transportType = (raw.transport as string) || 'stdio'
  const commandOrUrl = (raw.command_or_url as string) || ''

  let args: string[] = []
  try {
    const parsed = JSON.parse((raw.args as string) || '[]')
    args = Array.isArray(parsed) ? parsed : []
  } catch { /* ignore */ }

  let env: Record<string, string> = {}
  try {
    const parsed = JSON.parse((raw.env_vars as string) || '{}')
    env = (typeof parsed === 'object' && !Array.isArray(parsed)) ? parsed : {}
  } catch { /* ignore */ }

  return {
    id: raw.id as number,
    name: (raw.name as string) || '',
    description: (raw.description as string) || '',
    transport_type: transportType as McpServerInfo['transport_type'],
    command: transportType === 'stdio' ? commandOrUrl : '',
    url: transportType !== 'stdio' ? commandOrUrl : '',
    args,
    env,
    timeout_seconds: (raw.timeout_seconds as number) || 30,
    is_enabled: raw.is_enabled as boolean,
    tool_count: (raw.tool_count as number) || 0,
    status: (raw.status as string) || 'disconnected',
    last_error: (raw.last_error as string) || '',
    created_at: (raw.created_at as string) || '',
    updated_at: (raw.updated_at as string) || '',
  }
}

const parseJsonArray = (value: unknown): string[] => {
  if (Array.isArray(value)) return value.map(String).filter(Boolean)
  if (typeof value !== 'string' || !value) return []
  try {
    const parsed: unknown = JSON.parse(value)
    return Array.isArray(parsed) ? parsed.map(String).filter(Boolean) : []
  } catch {
    return []
  }
}

const parseJsonObject = (value: unknown): Record<string, unknown> => {
  if (value && typeof value === 'object' && !Array.isArray(value)) return value as Record<string, unknown>
  if (typeof value !== 'string' || !value) return {}
  try {
    const parsed: unknown = JSON.parse(value)
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed as Record<string, unknown> : {}
  } catch {
    return {}
  }
}

const stringifyArray = (value?: string[]): string =>
  JSON.stringify((value || []).map((item) => item.trim()).filter(Boolean))

const stringifyObject = (value?: Record<string, unknown>): string =>
  JSON.stringify(value || {})

function transformTool(raw: Record<string, unknown>): McpToolInfo {
  return {
    name: (raw.name as string) || '',
    description: (raw.description as string) || '',
    input_schema: parseJsonObject(raw.input_schema ?? raw.schema_json),
    server_id: (raw.server_id as number) || 0,
    server_name: (raw.server_name as string) || '',
  }
}

function transformPolicy(raw: Record<string, unknown>): McpToolPolicy {
  return {
    id: (raw.id as number) || 0,
    server_id: (raw.server_id as number) || 0,
    server_name: (raw.server_name as string) || '',
    tool_name: (raw.tool_name as string) || '',
    effect: ((raw.effect as string) || 'allow') as McpToolPolicy['effect'],
    risk_level: ((raw.risk_level as string) || 'low') as McpToolPolicy['risk_level'],
    require_confirmation: Boolean(raw.require_confirmation),
    allowed_roles: parseJsonArray(raw.allowed_roles ?? raw.allowed_roles_json),
    allowed_scopes: parseJsonArray(raw.allowed_scopes ?? raw.allowed_scopes_json),
    required_args: parseJsonArray(raw.required_args ?? raw.required_args_json),
    denied_args: parseJsonArray(raw.denied_args ?? raw.denied_args_json),
    arg_rules: parseJsonObject(raw.arg_rules ?? raw.arg_rules_json),
    redact_fields: parseJsonArray(raw.redact_fields ?? raw.redact_fields_json),
    rate_limit_window_seconds: (raw.rate_limit_window_seconds as number) || 0,
    rate_limit_max_calls: (raw.rate_limit_max_calls as number) || 0,
    is_enabled: raw.is_enabled as boolean,
    created_by_hr_id: (raw.created_by_hr_id as number) || 0,
    updated_by_hr_id: (raw.updated_by_hr_id as number) || 0,
    created_at: (raw.created_at as string) || '',
    updated_at: (raw.updated_at as string) || '',
  }
}

function transformLog(raw: Record<string, unknown>): McpCallLog {
  return {
    id: (raw.id as number) || 0,
    server_id: (raw.server_id as number) || 0,
    server_name: (raw.server_name as string) || '',
    tool_name: (raw.tool_name as string) || '',
    input_args: (raw.input_args as string) || (raw.args_json as string) || '',
    result: (raw.result as string) || (raw.result_content as string) || '',
    duration_ms: (raw.duration_ms as number) || 0,
    status: (raw.status as string) || ((raw.error_message || raw.error_msg) ? 'error' : 'success'),
    error_message: (raw.error_message as string) || (raw.error_msg as string) || '',
    called_by: (raw.called_by as string) || (raw.called_by_hr_id ? `HR #${raw.called_by_hr_id}` : ''),
    policy_id: (raw.policy_id as number) || 0,
    policy_decision: (raw.policy_decision as string) || '',
    policy_reason: (raw.policy_reason as string) || '',
    created_at: (raw.created_at as string) || '',
  }
}

// ── MCP Server CRUD ────────────────────────────────────────────────────────

export const listMcpServers = async (
  page = 1,
  pageSize = 20,
): Promise<PaginatedList<McpServerInfo>> => {
  const res: any = await request.get('/api/v1/hr/admin/mcp-servers', {
    params: { page, page_size: pageSize },
  })
  return {
    total: res.total || 0,
    list: (res.list || []).map(transformServer),
  }
}

export const createMcpServer = async (
  data: CreateMcpServerPayload,
): Promise<{ server: McpServerInfo }> => {
  const body: Record<string, unknown> = {
    name: data.name,
    description: data.description || '',
    transport: data.transport_type,
    command_or_url: data.transport_type === 'stdio' ? data.command : data.url,
    timeout_seconds: data.timeout_seconds ?? 30,
  }
  if (data.args?.length) body.args = JSON.stringify(data.args)
  if (data.env && Object.keys(data.env).length) body.env_vars = JSON.stringify(data.env)
  const res: any = await request.post('/api/v1/hr/admin/mcp-servers', body)
  return { server: transformServer(res.server || res) }
}

export const updateMcpServer = async (
  id: number,
  data: UpdateMcpServerPayload,
): Promise<{ server: McpServerInfo }> => {
  const body: Record<string, unknown> = {}
  if (data.name !== undefined) body.name = data.name
  if (data.description !== undefined) body.description = data.description
  if (data.transport_type !== undefined) {
    body.transport = data.transport_type
    body.command_or_url = data.transport_type === 'stdio' ? data.command : data.url
  } else if (data.command !== undefined || data.url !== undefined) {
    body.command_or_url = data.command ?? data.url
  }
  if (data.args !== undefined) body.args = JSON.stringify(data.args)
  if (data.env !== undefined) body.env_vars = JSON.stringify(data.env)
  if (data.timeout_seconds !== undefined) body.timeout_seconds = data.timeout_seconds
  if (data.is_enabled !== undefined) body.is_enabled = data.is_enabled
  const res: any = await request.put(`/api/v1/hr/admin/mcp-servers/${id}`, body)
  return { server: transformServer(res.server || res) }
}

export const deleteMcpServer = (id: number): Promise<void> =>
  request.delete(`/api/v1/hr/admin/mcp-servers/${id}`)

// ── Connection Test ────────────────────────────────────────────────────────

export const testMcpServerConnection = (
  id: number,
): Promise<McpTestResult> =>
  request.post(`/api/v1/hr/admin/mcp-servers/${id}/test`)

// ── Tools ──────────────────────────────────────────────────────────────────

export const listMcpServerTools = (
  serverId: number,
): Promise<{ tools: McpToolInfo[] }> =>
  request
    .get<{ list?: Record<string, unknown>[]; tools?: Record<string, unknown>[] }>(`/api/v1/hr/admin/mcp-servers/${serverId}/tools`)
    .then((res) => ({ tools: (res.tools || res.list || []).map(transformTool) }))

export const callMcpTool = (
  serverId: number,
  data: { tool_name: string; args: Record<string, unknown> },
): Promise<{
  result_content: string
  error_msg: string
  duration_ms: number
  policy_decision: string
  policy_reason: string
  policy_id: number
}> =>
  request.post(`/api/v1/hr/admin/mcp-servers/${serverId}/call-tool`, {
    tool_name: data.tool_name,
    args_json: JSON.stringify(data.args ?? {}),
  })

// ── Tool Policies ─────────────────────────────────────────────────────────

const policyBody = (
  data: CreateMcpToolPolicyPayload | UpdateMcpToolPolicyPayload,
): Record<string, unknown> => {
  const body: Record<string, unknown> = {}
  if (data.server_id !== undefined) body.server_id = data.server_id
  if (data.tool_name !== undefined) body.tool_name = data.tool_name
  if (data.effect !== undefined) body.effect = data.effect
  if (data.risk_level !== undefined) body.risk_level = data.risk_level
  if (data.require_confirmation !== undefined) {
    body.require_confirmation = data.require_confirmation
    body.require_confirmation_set = true
  }
  if (data.allowed_roles !== undefined) body.allowed_roles_json = stringifyArray(data.allowed_roles)
  if (data.allowed_scopes !== undefined) body.allowed_scopes_json = stringifyArray(data.allowed_scopes)
  if (data.required_args !== undefined) body.required_args_json = stringifyArray(data.required_args)
  if (data.denied_args !== undefined) body.denied_args_json = stringifyArray(data.denied_args)
  if (data.arg_rules !== undefined) body.arg_rules_json = stringifyObject(data.arg_rules)
  if (data.redact_fields !== undefined) body.redact_fields_json = stringifyArray(data.redact_fields)
  if (data.rate_limit_window_seconds !== undefined) {
    body.rate_limit_window_seconds = data.rate_limit_window_seconds
    body.rate_limit_window_seconds_set = true
  }
  if (data.rate_limit_max_calls !== undefined) {
    body.rate_limit_max_calls = data.rate_limit_max_calls
    body.rate_limit_max_calls_set = true
  }
  if (data.is_enabled !== undefined) {
    body.is_enabled = data.is_enabled
    body.is_enabled_set = true
  }
  return body
}

export const listMcpToolPolicies = async (
  query?: McpPolicyQuery,
): Promise<PaginatedList<McpToolPolicy>> => {
  const params: Record<string, number> = {}
  if (query?.page) params.page = query.page
  if (query?.page_size) params.page_size = query.page_size
  if (query?.server_id) params.server_id = query.server_id
  const res: any = await request.get('/api/v1/hr/admin/mcp-tool-policies', { params })
  return {
    total: res.total || 0,
    list: (res.list || []).map(transformPolicy),
  }
}

export const createMcpToolPolicy = async (
  data: CreateMcpToolPolicyPayload,
): Promise<{ policy: McpToolPolicy }> => {
  const res: any = await request.post('/api/v1/hr/admin/mcp-tool-policies', policyBody(data))
  return { policy: transformPolicy(res.policy || res) }
}

export const updateMcpToolPolicy = async (
  id: number,
  data: UpdateMcpToolPolicyPayload,
): Promise<{ policy: McpToolPolicy }> => {
  const res: any = await request.put(`/api/v1/hr/admin/mcp-tool-policies/${id}`, policyBody(data))
  return { policy: transformPolicy(res.policy || res) }
}

export const deleteMcpToolPolicy = (id: number): Promise<void> =>
  request.delete(`/api/v1/hr/admin/mcp-tool-policies/${id}`)

// ── Logs ───────────────────────────────────────────────────────────────────

export const listMcpServerLogs = (
  serverId: number,
  query?: McpLogQuery,
): Promise<PaginatedList<McpCallLog>> => {
  const params: Record<string, string | number> = {}
  if (query) {
    if (query.page) params.page = query.page
    if (query.page_size) params.page_size = query.page_size
    if (query.tool_name) params.tool_name = query.tool_name
    if (query.start_time) params.start_time = query.start_time
    if (query.end_time) params.end_time = query.end_time
  }
  return request
    .get<{ total?: number; list?: Record<string, unknown>[] }>(`/api/v1/hr/admin/mcp-servers/${serverId}/logs`, { params })
    .then((res) => ({
      total: res.total || 0,
      list: (res.list || []).map(transformLog),
    }))
}
