import request from './request'
import type { PaginatedList } from '@/types/domain'
import type {
  McpServerInfo,
  McpToolInfo,
  McpCallLog,
  CreateMcpServerPayload,
  UpdateMcpServerPayload,
  McpLogQuery,
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
  request.get(`/api/v1/hr/admin/mcp-servers/${serverId}/tools`)

export const callMcpTool = (
  serverId: number,
  data: { tool_name: string; args: Record<string, unknown> },
): Promise<{ result: unknown }> =>
  request.post(`/api/v1/hr/admin/mcp-servers/${serverId}/call`, data)

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
  return request.get(`/api/v1/hr/admin/mcp-servers/${serverId}/logs`, { params })
}
