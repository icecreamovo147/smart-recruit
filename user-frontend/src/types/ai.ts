// ---- Candidate AI Types ----

export interface CandidateChatMessage {
  role: 'user' | 'assistant'
  content: string
  pending?: boolean
  failed?: boolean
  waitingText?: string
  suggestedQuestions?: string[]
}

export interface CandidateSession {
  session_id: number
  title: string
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
  reply?: string
  created_at?: string
  action?: string
  action_payload?: string
  candidate_options?: string
  suggested_questions?: string[] | string
  suggestedQuestions?: string[] | string
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

export interface RecommendedJob {
  job_id: number
  title: string
  department?: string
  location?: string
  salary_range?: string
  status: number
  has_applied: boolean
  match_score?: number
  reasons: string[]
}

export type CandidateAIAction =
  | { action: 'recommend_jobs'; jobs: RecommendedJob[] }
  | { action: 'view_job'; job_id: number }
