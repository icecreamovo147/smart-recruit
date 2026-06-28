// ---- AI Chat Types ----

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  created_at?: string
  model_id?: number
  model_name?: string
  pending?: boolean
  failed?: boolean
  waitingText?: string
  candidateOptions?: CandidateOption[]
}

export interface CandidateOption {
  application_id: number
  candidate_name: string
  job_title: string
  masked_phone: string
  round_no: number
  status_text?: string
  is_current?: number
  applied_at?: string
}

export interface Session {
  id: number
  title: string
  application_id: number
  created_at?: string
  updated_at?: string
}

export interface StreamPayload {
  delta?: string
  done?: boolean
  code?: number
  msg?: string
  request_id?: string
  session_id?: number
  candidate_options?: string
  action?: string
  action_status?: number
  application_id?: number
  candidate_name?: string
  job_title?: string
  status?: number
  created_at?: string
  // Phase 4: streaming UX status events
  event_type?: string // thinking | tool_calling | tool_done | generating | timeout_warning | partial_done | done | error | model_info | agent_run_started | model_selected | planning | capability_selected | fallback | agent_run_done
  event_message?: string
  error_type?: string
  tool_name?: string
  model_name?: string
}

export interface StreamHandlers {
  onDelta?: (delta: string, payload: StreamPayload) => void
  onDone?: (payload: StreamPayload) => void
  onStatus?: (eventType: string, eventMessage: string, payload: StreamPayload) => void
  onError?: (errorType: string, errorMessage: string, payload: StreamPayload) => void
}

export interface StreamErrorPayload {
  code: number
  msg: string
  request_id?: string
}

export interface ChatSessionListItem {
  session_id: number
  title: string
  application_id: number
  created_at: string
  updated_at: string
}

// ---- Agent Tool Trace Types ----

export interface ToolTraceItem {
  id: number
  session_id: number
  tool_name: string
  args_json: string
  result_content: string
  duration_ms: number
  error_msg: string
  created_at: string
}

export interface AgentRunStepItem {
  id: number
  run_id: number
  step_index: number
  step_type: string
  capability_source: string
  capability_key: string
  tool_name: string
  input_json: string
  output_json: string
  status: string
  duration_ms: number
  error_message: string
  started_at: string
  completed_at: string
  created_at: string
}

export interface AgentRunItem {
  id: number
  session_id: number
  message_id: number
  history_id: number
  hr_id: number
  agent_type: string
  agent_id: number
  agent_name: string
  model_id: number
  model_name: string
  status: string
  plan_json: string
  final_answer: string
  error_type: string
  error_message: string
  started_at: string
  completed_at: string
  created_at: string
  steps: AgentRunStepItem[]
}
