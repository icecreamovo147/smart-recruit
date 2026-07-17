import type { StreamEnvelope } from '../types/api'

export interface MessageEventLike {
  data: string
}

export interface EventSourceLike {
  onopen: ((event: Event) => void) | null
  onerror: ((event: Event) => void) | null
  addEventListener(type: string, listener: (event: MessageEventLike) => void): void
  close(): void
}

export type EventSourceFactory = (url: string) => EventSourceLike

export function buildStreamURL(services: readonly string[], tail = 300): string {
  const params = new URLSearchParams()
  for (const service of services) params.append('service', service)
  params.set('tail', String(tail))
  return `/api/v1/logs/stream?${params.toString()}`
}

export function decodeStreamEnvelope(raw: string): StreamEnvelope {
  const parsed: unknown = JSON.parse(raw)
  if (!isRecord(parsed) || typeof parsed.type !== 'string') {
    throw new Error('Malformed stream envelope')
  }
  switch (parsed.type) {
    case 'snapshot_start':
    case 'snapshot_end':
    case 'heartbeat':
      return { type: parsed.type, event_id: optionalNumber(parsed.event_id) }
    case 'log':
      if (typeof parsed.service !== 'string' || !isRecord(parsed.payload)) throw new Error('Malformed log envelope')
      return {
        type: 'log',
        event_id: optionalNumber(parsed.event_id),
        service: parsed.service,
        payload: parsed.payload as StreamEnvelope & never,
      }
    case 'reset':
      return {
        type: 'reset',
        event_id: optionalNumber(parsed.event_id),
        service: typeof parsed.service === 'string' ? parsed.service : undefined,
        recoverable: Boolean(parsed.recoverable),
        payload: parsed.payload,
      }
    case 'dropped':
      if (typeof parsed.dropped !== 'number') throw new Error('Malformed dropped envelope')
      return { type: 'dropped', event_id: optionalNumber(parsed.event_id), dropped: parsed.dropped }
    case 'status':
      return {
        type: 'status',
        event_id: optionalNumber(parsed.event_id),
        service: typeof parsed.service === 'string' ? parsed.service : undefined,
        payload: parsed.payload,
      }
    default:
      throw new Error(`Unsupported stream envelope: ${parsed.type}`)
  }
}

function optionalNumber(value: unknown): number | undefined {
  return typeof value === 'number' ? value : undefined
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
