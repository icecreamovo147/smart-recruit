/**
 * Durable HR Agent run types aligned with web-gin gateway JSON
 * (snake_case field names after interceptor unwrap of `data`).
 */

export type AgentRunStatus =
  | 'queued'
  | 'planning'
  | 'running'
  | 'waiting_confirmation'
  | 'cancel_requested'
  | 'succeeded'
  | 'partial'
  | 'failed'
  | 'canceled'

export type AgentRunEventType =
  | 'run.created'
  | 'run.status_changed'
  | 'assistant.delta'
  | 'assistant.snapshot'
  | 'process.delta'
  | 'process.snapshot'
  | 'tool.started'
  | 'tool.finished'
  | 'confirmation.required'
  | 'confirmation.accepted'
  | 'run.result'
  | 'run.error'
  | 'run.completed'
  | 'run.canceled'
  | 'run.heartbeat'

export const AGENT_RUN_TERMINAL_STATUSES: readonly AgentRunStatus[] = [
  'succeeded',
  'partial',
  'failed',
  'canceled',
] as const

export const AGENT_RUN_ACTIVE_STATUSES: readonly AgentRunStatus[] = [
  'queued',
  'planning',
  'running',
  'waiting_confirmation',
  'cancel_requested',
] as const

export function isTerminalAgentRunStatus(status: string | null | undefined): boolean {
  return AGENT_RUN_TERMINAL_STATUSES.includes((status || '') as AgentRunStatus)
}

export function isActiveAgentRunStatus(status: string | null | undefined): boolean {
  return AGENT_RUN_ACTIVE_STATUSES.includes((status || '') as AgentRunStatus)
}

/** Reuses existing context usage shape from chat types when present. */
export interface AgentRunContextUsage {
  model_id?: number
  model_name?: string
  context_window_tokens?: number
  max_output_tokens?: number
  prompt_tokens_estimated?: number
  prompt_tokens_actual?: number
  completion_tokens_actual?: number
  total_tokens_actual?: number
  remaining_tokens_estimated?: number
  usage_ratio?: number
  estimated?: boolean
  source?: string
  stage?: string
  input_budget_tokens?: number
  safety_margin_tokens?: number
  budget_usage_ratio?: number
  budget_status?: string
  included_message_count?: number
  omitted_message_count?: number
  summary_applied?: boolean
  breakdown?: {
    system_prompt_tokens?: number
    recent_message_tokens?: number
    summary_tokens?: number
    memory_tokens?: number
    current_message_tokens?: number
    skill_tokens?: number
    tool_result_tokens?: number
    tool_schema_tokens?: number
    protocol_overhead_tokens?: number
  }
}

export interface AgentRunResultMetadata {
  action?: string
  application_id?: number
  action_status?: number
  candidate_name?: string
  job_title?: string
  status?: number
  candidate_options?: string
  suggested_questions?: string[]
  context_usage?: AgentRunContextUsage | null
  error_type?: string
  error_message?: string
  raw_json?: string
}

export interface AgentRunSkillCandidate {
  id?: number
  name?: string
  display_name?: string
  reason?: string
  score?: number
  priority?: number
  category?: string
  scenario?: string
  risk_level?: string
  recommended?: boolean
  vector_score?: number
  lexical_score?: number
  metadata_score?: number
  relevance_score?: number
  business_boost?: number
  final_rank_score?: number
  relevance_mode?: string
  pool_rank?: number
  ranking_confidence?: string
}

export interface AgentRunConfirmation {
  required?: boolean
  reason?: string
  candidates?: AgentRunSkillCandidate[]
  recommended_agent_skill_ids?: number[]
  user_message_id?: number
  raw_json?: string
}

export interface AgentRunSnapshot {
  run_id: number
  session_id: number
  hr_id?: number
  client_request_id?: string
  message_id?: number
  history_id?: number
  status: AgentRunStatus | string
  assistant_text?: string
  process_text?: string
  result_metadata?: AgentRunResultMetadata | null
  confirmation_request?: AgentRunConfirmation | null
  option_context_json?: string
  last_event_seq: number
  error_type?: string
  error_message?: string
  model_id?: number
  model_name?: string
  agent_type?: string
  agent_id?: number
  agent_name?: string
  started_at?: string
  completed_at?: string
  cancel_requested_at?: string
  canceled_at?: string
  created_at?: string
  updated_at?: string
}

export interface AgentRunEvent {
  run_id: number
  seq: number
  event_type: AgentRunEventType | string
  payload_json?: string
  event_message?: string
  display_message?: string
  display_source?: string
  step_key?: string
  step_purpose?: string
  tool_group?: string
  status?: string
  delta?: string
  snapshot_text?: string
  result_metadata?: AgentRunResultMetadata | null
  confirmation?: AgentRunConfirmation | null
  tool_name?: string
  error_type?: string
  error_message?: string
  created_at?: string
  /** Present on SSE transport errors, not durable run events. */
  code?: number
  msg?: string
  done?: boolean
  request_id?: string
}

export interface CreateAgentRunRequest {
  session_id: number
  client_request_id?: string
  message?: string
  action_type?: string
  action_payload_json?: string
  application_id?: number
  model_id?: number
  skill_capability_keys?: string[]
  agent_skill_ids?: number[]
  agent_skill_selection_confirmed?: boolean
  agent_skill_selection_message_id?: number
}

export interface CreateAgentRunResponse {
  run: AgentRunSnapshot
  idempotent_replay?: boolean
}

export interface GetAgentRunResponse {
  run: AgentRunSnapshot
}

export interface GetActiveAgentRunResponse {
  run: AgentRunSnapshot | null
  has_active_run: boolean
}

export interface CancelAgentRunRequest {
  client_request_id?: string
}

export interface CancelAgentRunResponse {
  run: AgentRunSnapshot
}

export interface ConfirmAgentRunRequest {
  client_request_id?: string
  agent_skill_ids?: number[]
  agent_skill_selection_confirmed?: boolean
  agent_skill_selection_message_id?: number
  confirmation_payload_json?: string
}

export interface ConfirmAgentRunResponse {
  run: AgentRunSnapshot
}

export interface AgentRunEventHandlers {
  onEvent?: (event: AgentRunEvent) => void
  onError?: (error: { code?: number; message: string; event?: AgentRunEvent }) => void
  onDone?: () => void
}
