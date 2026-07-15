import type { ConnectionStatus, LogLevel, LogRecord, ServiceStatus, StreamEnvelope } from '../types/api'

export const LOG_BUFFER_LIMIT = 10_000

export interface Filters {
  services: Set<string>
  levels: Set<LogLevel>
  text: string
  correlation: string
}

export interface LogViewerState {
  services: ServiceStatus[]
  records: LogRecord[]
  seenEventIds: Set<number>
  dropped: number
  resets: number
  protocolErrors: number
  connectionStatus: ConnectionStatus
  renderPaused: boolean
  followLatest: boolean
  unseen: number
  filters: Filters
}

export function createInitialState(): LogViewerState {
  return {
    services: [],
    records: [],
    seenEventIds: new Set(),
    dropped: 0,
    resets: 0,
    protocolErrors: 0,
    connectionStatus: 'idle',
    renderPaused: false,
    followLatest: true,
    unseen: 0,
    filters: {
      services: new Set(),
      levels: new Set(),
      text: '',
      correlation: '',
    },
  }
}

export function applyEnvelope(state: LogViewerState, envelope: StreamEnvelope): LogViewerState {
  if ('event_id' in envelope && typeof envelope.event_id === 'number') {
    if (state.seenEventIds.has(envelope.event_id)) return state
  }
  switch (envelope.type) {
    case 'log':
      return appendRecord(markSeen(state, envelope.event_id), envelope.payload)
    case 'reset':
      return { ...markSeen(state, envelope.event_id), records: [], resets: state.resets + 1, unseen: 0 }
    case 'dropped':
      return { ...markSeen(state, envelope.event_id), dropped: state.dropped + envelope.dropped }
    case 'snapshot_start':
      return { ...markSeen(state, envelope.event_id), connectionStatus: 'connected' }
    case 'snapshot_end':
    case 'heartbeat':
    case 'status':
      return markSeen(state, envelope.event_id)
  }
}

export function markMalformedEvent(state: LogViewerState): LogViewerState {
  return { ...state, protocolErrors: state.protocolErrors + 1 }
}

export function setConnectionStatus(state: LogViewerState, connectionStatus: ConnectionStatus): LogViewerState {
  return { ...state, connectionStatus }
}

export function setRenderPaused(state: LogViewerState, renderPaused: boolean): LogViewerState {
  return { ...state, renderPaused }
}

export function setFollowLatest(state: LogViewerState, followLatest: boolean): LogViewerState {
  return { ...state, followLatest, unseen: followLatest ? 0 : state.unseen }
}

export function setFilters(state: LogViewerState, filters: Partial<Filters>): LogViewerState {
  return {
    ...state,
    filters: {
      services: filters.services ?? state.filters.services,
      levels: filters.levels ?? state.filters.levels,
      text: filters.text ?? state.filters.text,
      correlation: filters.correlation ?? state.filters.correlation,
    },
  }
}

export function clearRecords(state: LogViewerState): LogViewerState {
  return { ...state, records: [], seenEventIds: new Set(), unseen: 0 }
}

export function selectFilteredRecords(state: LogViewerState): LogRecord[] {
  const text = state.filters.text.trim().toLowerCase()
  const correlation = state.filters.correlation.trim()
  return state.records.filter((record) => {
    if (state.filters.services.size > 0 && !state.filters.services.has(record.service)) return false
    if (state.filters.levels.size > 0 && !state.filters.levels.has(record.level)) return false
    if (text && !record.message.toLowerCase().includes(text) && !record.lines.join('\n').toLowerCase().includes(text)) return false
    if (correlation && record.request_id !== correlation && record.trace_id !== correlation && record.span_id !== correlation) {
      return false
    }
    return true
  })
}

function appendRecord(state: LogViewerState, record: LogRecord): LogViewerState {
  const records = state.records.length >= LOG_BUFFER_LIMIT ? state.records.slice(state.records.length - LOG_BUFFER_LIMIT + 1) : state.records.slice()
  const evicted = state.records.length >= LOG_BUFFER_LIMIT ? state.records.length - records.length : 0
  const previousID = records.at(-1)?.id ?? 0
  records.push(record.id > 0 ? record : { ...record, id: previousID + 1 })
  const shouldCountUnseen = state.renderPaused || !state.followLatest
  return {
    ...state,
    records,
    dropped: state.dropped + evicted,
    unseen: shouldCountUnseen ? state.unseen + 1 : state.unseen,
  }
}

function markSeen(state: LogViewerState, eventID: number | undefined): LogViewerState {
  if (typeof eventID !== 'number') return state
  const seenEventIds = new Set(state.seenEventIds)
  seenEventIds.add(eventID)
  return { ...state, seenEventIds }
}
