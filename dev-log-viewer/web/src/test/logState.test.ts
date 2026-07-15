import { describe, expect, it } from 'vitest'
import {
  LOG_BUFFER_LIMIT,
  applyEnvelope,
  createInitialState,
  markMalformedEvent,
  selectFilteredRecords,
  setFilters,
  setFollowLatest,
  setRenderPaused,
} from '../state/logState'
import type { LogRecord } from '../types/api'

describe('logState reducer', () => {
  it('deduplicates event ids and bounds the log ring', () => {
    let state = createInitialState()
    for (let index = 0; index < LOG_BUFFER_LIMIT + 2; index += 1) {
      state = applyEnvelope(state, { type: 'log', event_id: index + 1, service: 'svc', payload: record(index) })
    }
    expect(state.records).toHaveLength(LOG_BUFFER_LIMIT)
    expect(state.records[0]?.id).toBe(2)
    expect(state.dropped).toBe(2)

    const before = state
    state = applyEnvelope(state, { type: 'log', event_id: 2, service: 'svc', payload: record(999) })
    expect(state).toBe(before)
  })

  it('applies AND filters without mutating the source buffer', () => {
    let state = createInitialState()
    state = applyEnvelope(state, { type: 'log', event_id: 1, service: 'api', payload: record(1, 'api', 'ERROR', 'failed request', 'req-1') })
    state = applyEnvelope(state, { type: 'log', event_id: 2, service: 'web', payload: record(2, 'web', 'INFO', 'ready', 'req-2') })
    state = setFilters(state, {
      services: new Set(['api']),
      levels: new Set(['ERROR']),
      text: 'failed',
      correlation: 'req-1',
    })

    const filtered = selectFilteredRecords(state)
    expect(filtered.map((item) => item.id)).toEqual([1])
    expect(state.records).toHaveLength(2)
  })

  it('handles reset, dropped, malformed, pause, follow, and unseen independently', () => {
    let state = createInitialState()
    state = setFollowLatest(state, false)
    state = setRenderPaused(state, true)
    state = applyEnvelope(state, { type: 'log', event_id: 1, service: 'svc', payload: record(1) })
    expect(state.unseen).toBe(1)

    state = applyEnvelope(state, { type: 'dropped', event_id: 2, dropped: 7 })
    expect(state.dropped).toBe(7)
    state = markMalformedEvent(state)
    expect(state.protocolErrors).toBe(1)
    state = applyEnvelope(state, { type: 'reset', event_id: 3, recoverable: false })
    expect(state.records).toHaveLength(0)
    expect(state.resets).toBe(1)
    expect(state.renderPaused).toBe(true)
    expect(state.followLatest).toBe(false)
  })
})

function record(
  id: number,
  service = 'svc',
  level: LogRecord['level'] = 'INFO',
  message = `message ${id}`,
  requestID = `req-${id}`,
): LogRecord {
  return {
    id,
    generation: 0,
    service,
    observed_at: '2026-07-15T09:00:00Z',
    level,
    message,
    lines: [message],
    request_id: requestID,
  }
}
