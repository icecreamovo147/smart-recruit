import { describe, expect, it } from 'vitest'

import {
  formatPlainDate,
  formatShanghaiDateTime,
  formatUnixDateTime,
  parseBusinessDateTime,
  toShanghaiRFC3339,
} from '@shared/utils/format'

describe('shared UTC+8 business time contract', () => {
  it('renders absolute instants in Shanghai regardless of host timezone', () => {
    expect(formatShanghaiDateTime('2026-07-21T10:00:00Z')).toBe('2026-07-21 18:00:00')
    expect(formatShanghaiDateTime('2026-07-21T18:00:00+08:00')).toBe('2026-07-21 18:00:00')
    expect(formatUnixDateTime(Date.parse('2026-07-21T10:00:00Z'))).toBe('2026-07-21 18:00:00')
  })

  it('interprets timezone-free DATETIME values as Shanghai wall-clock time', () => {
    expect(parseBusinessDateTime('2026-07-21 18:00:00').toISOString()).toBe('2026-07-21T10:00:00.000Z')
    expect(toShanghaiRFC3339('2026-07-21 18:00')).toBe('2026-07-21T18:00:00+08:00')
  })

  it('keeps plain dates out of timezone conversion', () => {
    expect(formatPlainDate('2026-07-21')).toBe('2026-07-21')
  })
})
