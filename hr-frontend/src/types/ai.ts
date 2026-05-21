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
}

export interface StreamHandlers {
  onDelta?: (delta: string, payload: StreamPayload) => void
  onDone?: (payload: StreamPayload) => void
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
