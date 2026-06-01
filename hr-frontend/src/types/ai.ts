// ---- AI Chat Types ----

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
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
  event_type?: string // thinking | tool_calling | tool_done | generating | timeout_warning | partial_done | done | error
  event_message?: string
  error_type?: string
  tool_name?: string
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
