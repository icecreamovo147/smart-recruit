export type ServiceGroup = 'backend' | 'gateway' | 'frontend'
export type ProcessState = 'running' | 'offline' | 'unknown'
export type LogState = 'ready' | 'missing' | 'unreadable'
export type LogLevel = 'DEBUG' | 'INFO' | 'WARN' | 'ERROR' | 'DPANIC' | 'PANIC' | 'FATAL' | 'UNKNOWN'

export interface ServiceStatus {
  id: string
  name: string
  group: ServiceGroup
  port: number
  process_state: ProcessState
  log_state: LogState
  log_size_bytes: number
  log_updated_at?: string
}

export interface ServicesResponse {
  services: ServiceStatus[]
}

export interface LogRecord {
  id: number
  generation: number
  service: string
  observed_at: string
  source_time?: string
  level: LogLevel
  caller?: string
  message: string
  lines: string[]
  fields?: Record<string, string>
  request_id?: string
  trace_id?: string
  span_id?: string
  truncated?: boolean
}

export type StreamEnvelope =
  | { type: 'snapshot_start'; event_id?: number }
  | { type: 'log'; event_id?: number; service: string; payload: LogRecord }
  | { type: 'snapshot_end'; event_id?: number }
  | { type: 'reset'; event_id?: number; service?: string; recoverable: boolean; payload?: unknown }
  | { type: 'dropped'; event_id?: number; dropped: number }
  | { type: 'heartbeat'; event_id?: number }
  | { type: 'status'; event_id?: number; service?: string; payload?: unknown }

export type ConnectionStatus = 'idle' | 'connecting' | 'connected' | 'reconnecting' | 'disconnected'
