// ── MCP Server ────────────────────────────────────────────────────────────

export type McpTransportType = 'stdio' | 'sse' | 'http'

export interface McpServerInfo {
  id: number
  name: string
  description: string
  transport_type: McpTransportType
  command: string           // For stdio: the command to run
  url: string               // For sse/http: the endpoint URL
  args: string[]            // Command arguments (stdio)
  env: Record<string, string>  // Environment variables (stdio)
  timeout_seconds: number
  is_enabled: boolean
  tool_count: number
  status: string            // connected / disconnected / error
  last_error: string
  created_at: string
  updated_at: string
}

// ── MCP Tools ──────────────────────────────────────────────────────────────

export interface McpToolInfo {
  name: string
  description: string
  input_schema: Record<string, unknown>
  server_id: number
  server_name: string
}

// ── Tool Call Log ──────────────────────────────────────────────────────────

export interface McpCallLog {
  id: number
  server_id: number
  server_name: string
  tool_name: string
  input_args: string          // JSON string of input arguments
  result: string              // JSON string of result
  duration_ms: number
  status: string              // success / error
  error_message: string
  called_by: string
  created_at: string
}

// ── Payloads ───────────────────────────────────────────────────────────────

export interface CreateMcpServerPayload {
  name: string
  description?: string
  transport_type: McpTransportType
  command?: string
  url?: string
  args?: string[]
  env?: Record<string, string>
  timeout_seconds?: number
}

export interface UpdateMcpServerPayload {
  name?: string
  description?: string
  transport_type?: McpTransportType
  command?: string
  url?: string
  args?: string[]
  env?: Record<string, string>
  timeout_seconds?: number
  is_enabled?: boolean
}

// ── Log query params ──────────────────────────────────────────────────────

export interface McpLogQuery {
  page?: number
  page_size?: number
  tool_name?: string
  start_time?: string
  end_time?: string
}

// ── Connection test result ─────────────────────────────────────────────────

export interface McpTestResult {
  success: boolean
  message: string
  tools_found: number
  duration_ms: number
}
