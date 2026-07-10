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

// ── Tool Policy ───────────────────────────────────────────────────────────

export type McpPolicyEffect = 'allow' | 'deny'
export type McpPolicyRiskLevel = 'low' | 'medium' | 'high' | 'critical'

export interface McpToolPolicy {
  id: number
  server_id: number
  server_name?: string
  tool_name: string
  effect: McpPolicyEffect
  risk_level: McpPolicyRiskLevel
  require_confirmation: boolean
  allowed_roles: string[]
  allowed_scopes: string[]
  required_args: string[]
  denied_args: string[]
  arg_rules: Record<string, unknown>
  redact_fields: string[]
  rate_limit_window_seconds: number
  rate_limit_max_calls: number
  is_enabled: boolean
  created_by_hr_id: number
  updated_by_hr_id: number
  created_at: string
  updated_at: string
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
  policy_id: number
  policy_decision: string
  policy_reason: string
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

export interface CreateMcpToolPolicyPayload {
  server_id: number
  tool_name: string
  effect: McpPolicyEffect
  risk_level: McpPolicyRiskLevel
  require_confirmation: boolean
  allowed_roles?: string[]
  allowed_scopes?: string[]
  required_args?: string[]
  denied_args?: string[]
  arg_rules?: Record<string, unknown>
  redact_fields?: string[]
  rate_limit_window_seconds?: number
  rate_limit_max_calls?: number
  is_enabled?: boolean
}

export interface UpdateMcpToolPolicyPayload extends Partial<CreateMcpToolPolicyPayload> {}

export interface McpPolicyQuery {
  page?: number
  page_size?: number
  server_id?: number
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
