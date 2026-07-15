import { describe, expect, it } from 'vitest'
import { buildStreamURL, decodeStreamEnvelope, type EventSourceLike, type MessageEventLike } from '../api/stream'
import { LogStreamController } from '../hooks/useLogStream'
import type { ConnectionStatus, StreamEnvelope } from '../types/api'

describe('stream api and controller', () => {
  it('builds stream URLs and decodes envelopes', () => {
    expect(buildStreamURL(['b', 'a'], 10)).toBe('/api/v1/logs/stream?service=b&service=a&tail=10')
    const envelope = decodeStreamEnvelope('{"type":"dropped","dropped":3,"event_id":9}')
    expect(envelope).toEqual({ type: 'dropped', dropped: 3, event_id: 9 })
    expect(() => decodeStreamEnvelope('{"type":"log"}')).toThrow()
  })

  it('keeps a single EventSource and closes old connections', () => {
    const sources: MockEventSource[] = []
    const envelopes: StreamEnvelope[] = []
    const malformed: string[] = []
    const statuses: ConnectionStatus[] = []
    const controller = new LogStreamController({
      createEventSource: (url) => {
        const source = new MockEventSource(url)
        sources.push(source)
        return source
      },
      onEnvelope: (envelope) => envelopes.push(envelope),
      onMalformed: () => malformed.push('bad'),
      onStatus: (status) => statuses.push(status),
    })

    controller.connect(['api'], 5)
    sources[0]?.onopen?.()
    sources[0]?.emit('log', '{"type":"log","service":"api","payload":{"id":1,"generation":0,"service":"api","observed_at":"now","level":"INFO","message":"ok","lines":["ok"]}}')
    sources[0]?.emit('log', '{bad')
    controller.connect(['web'], 5)
    controller.close()

    expect(sources).toHaveLength(2)
    expect(sources[0]?.closed).toBe(true)
    expect(sources[1]?.closed).toBe(true)
    expect(envelopes).toHaveLength(1)
    expect(malformed).toHaveLength(1)
    expect(statuses).toEqual(['connecting', 'connected', 'disconnected', 'connecting', 'disconnected'])
  })
})

class MockEventSource implements EventSourceLike {
  onopen: (() => void) | null = null
  onerror: (() => void) | null = null
  closed = false
  listeners = new Map<string, (event: MessageEventLike) => void>()

  constructor(readonly url: string) {}

  addEventListener(type: string, listener: (event: MessageEventLike) => void): void {
    this.listeners.set(type, listener)
  }

  close(): void {
    this.closed = true
  }

  emit(type: string, data: string): void {
    this.listeners.get(type)?.({ data })
  }
}
