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

// ── MCP Server CRUD ────────────────────────────────────────────────────────

export const listMcpServers = (
  page = 1,
  pageSize = 20,
): Promise<PaginatedList<McpServerInfo>> => {
  return request.get('/api/v1/hr/admin/mcp-servers', {
    params: { page, page_size: pageSize },
  })
}

export const createMcpServer = (
  data: CreateMcpServerPayload,
): Promise<{ server: McpServerInfo }> =>
  request.post('/api/v1/hr/admin/mcp-servers', data)

export const updateMcpServer = (
  id: number,
  data: UpdateMcpServerPayload,
): Promise<{ server: McpServerInfo }> =>
  request.put(`/api/v1/hr/admin/mcp-servers/${id}`, data)

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
