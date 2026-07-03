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
import { debugLog } from '@/utils/debugLog'

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
  debugLog.mcp.info('listMcpServers_started', { page, page_size: pageSize })
  const res: any = await request.get('/api/v1/hr/admin/mcp-servers', {
    params: { page, page_size: pageSize },
  })
  const list = (res.list || []).map(transformServer)
  debugLog.mcp.info('listMcpServers_succeeded', { total: res.total || 0, returned: list.length })
  return {
    total: res.total || 0,
    list,
  }
}

export const createMcpServer = async (
  data: CreateMcpServerPayload,
): Promise<{ server: McpServerInfo }> => {
  debugLog.mcp.info('createMcpServer_started', { name: data.name, transport_type: data.transport_type })
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
  const server = transformServer(res.server || res)
  debugLog.mcp.info('createMcpServer_succeeded', { server_id: server.id, name: server.name })
  return { server }
}

export const updateMcpServer = async (
  id: number,
  data: UpdateMcpServerPayload,
): Promise<{ server: McpServerInfo }> => {
  debugLog.mcp.info('updateMcpServer_started', { server_id: id })
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
  const server = transformServer(res.server || res)
  debugLog.mcp.info('updateMcpServer_succeeded', { server_id: id })
  return { server }
}

export const deleteMcpServer = async (id: number): Promise<void> => {
  debugLog.mcp.info('deleteMcpServer_started', { server_id: id })
  await request.delete(`/api/v1/hr/admin/mcp-servers/${id}`)
  debugLog.mcp.info('deleteMcpServer_succeeded', { server_id: id })
}

// ── Connection Test ────────────────────────────────────────────────────────

export const testMcpServerConnection = async (
  id: number,
): Promise<McpTestResult> => {
  debugLog.mcp.info('testMcpServerConnection_started', { server_id: id })
  const result: McpTestResult = await request.post(`/api/v1/hr/admin/mcp-servers/${id}/test`)
  debugLog.mcp.info('testMcpServerConnection_succeeded', { server_id: id, success: result.success, tools_found: result.tools_found, duration_ms: result.duration_ms })
  return result
}

// ── Tools ──────────────────────────────────────────────────────────────────

export const listMcpServerTools = async (
  serverId: number,
): Promise<{ tools: McpToolInfo[] }> => {
  debugLog.mcp.info('listMcpServerTools_started', { server_id: serverId })
  const res = await request
    .get<{ list?: Record<string, unknown>[]; tools?: Record<string, unknown>[] }>(`/api/v1/hr/admin/mcp-servers/${serverId}/tools`)
  const tools = (res.tools || res.list || []).map(transformTool)
  debugLog.mcp.info('listMcpServerTools_succeeded', { server_id: serverId, tool_count: tools.length })
  return { tools }
}

export const callMcpTool = async (
  serverId: number,
  data: { tool_name: string; args: Record<string, unknown> },
): Promise<{
  result_content: string
  error_msg: string
  duration_ms: number
  policy_decision: string
  policy_reason: string
  policy_id: number
}> => {
  debugLog.mcp.info('callMcpTool_started', { server_id: serverId, tool_name: data.tool_name })
  const result: any = await request.post(`/api/v1/hr/admin/mcp-servers/${serverId}/call-tool`, {
    tool_name: data.tool_name,
    args_json: JSON.stringify(data.args ?? {}),
  })
  debugLog.mcp.info('callMcpTool_finished', {
    server_id: serverId,
    tool_name: data.tool_name,
    duration_ms: result.duration_ms,
    policy_decision: result.policy_decision,
    has_error: !!result.error_msg,
  })
  return result
}

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
  debugLog.mcp.info('listMcpToolPolicies_started', { server_id: query?.server_id, page: query?.page })
  const params: Record<string, number> = {}
  if (query?.page) params.page = query.page
  if (query?.page_size) params.page_size = query.page_size
  if (query?.server_id) params.server_id = query.server_id
  const res: any = await request.get('/api/v1/hr/admin/mcp-tool-policies', { params })
  const list = (res.list || []).map(transformPolicy)
  debugLog.mcp.info('listMcpToolPolicies_succeeded', { total: res.total || 0, returned: list.length })
  return {
    total: res.total || 0,
    list,
  }
}

export const createMcpToolPolicy = async (
  data: CreateMcpToolPolicyPayload,
): Promise<{ policy: McpToolPolicy }> => {
  debugLog.mcp.info('createMcpToolPolicy_started', { server_id: data.server_id, tool_name: data.tool_name, effect: data.effect })
  const res: any = await request.post('/api/v1/hr/admin/mcp-tool-policies', policyBody(data))
  const policy = transformPolicy(res.policy || res)
  debugLog.mcp.info('createMcpToolPolicy_succeeded', { policy_id: policy.id, tool_name: policy.tool_name })
  return { policy }
}

export const updateMcpToolPolicy = async (
  id: number,
  data: UpdateMcpToolPolicyPayload,
): Promise<{ policy: McpToolPolicy }> => {
  debugLog.mcp.info('updateMcpToolPolicy_started', { policy_id: id })
  const res: any = await request.put(`/api/v1/hr/admin/mcp-tool-policies/${id}`, policyBody(data))
  const policy = transformPolicy(res.policy || res)
  debugLog.mcp.info('updateMcpToolPolicy_succeeded', { policy_id: id })
  return { policy }
}

export const deleteMcpToolPolicy = async (id: number): Promise<void> => {
  debugLog.mcp.info('deleteMcpToolPolicy_started', { policy_id: id })
  await request.delete(`/api/v1/hr/admin/mcp-tool-policies/${id}`)
  debugLog.mcp.info('deleteMcpToolPolicy_succeeded', { policy_id: id })
}

// ── Logs ───────────────────────────────────────────────────────────────────

export const listMcpServerLogs = async (
  serverId: number,
  query?: McpLogQuery,
): Promise<PaginatedList<McpCallLog>> => {
  debugLog.mcp.info('listMcpServerLogs_started', { server_id: serverId, tool_name: query?.tool_name })
  const params: Record<string, string | number> = {}
  if (query) {
    if (query.page) params.page = query.page
    if (query.page_size) params.page_size = query.page_size
    if (query.tool_name) params.tool_name = query.tool_name
    if (query.start_time) params.start_time = query.start_time
    if (query.end_time) params.end_time = query.end_time
  }
  const res = await request
    .get<{ total?: number; list?: Record<string, unknown>[] }>(`/api/v1/hr/admin/mcp-servers/${serverId}/logs`, { params })
  const list = (res.list || []).map(transformLog)
  debugLog.mcp.info('listMcpServerLogs_succeeded', { server_id: serverId, total: res.total || 0, returned: list.length })
  return { total: res.total || 0, list }
}
