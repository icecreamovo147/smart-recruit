import { describe, expect, it } from 'vitest'
import { APP_TITLE, getPlaceholderStatus } from '../App'

describe('dev-log-viewer scaffold', () => {
  it('exposes canonical UI metadata', () => {
    expect(APP_TITLE).toBe('Dev Log Viewer')
    expect(getPlaceholderStatus()).toBe('Canonical UI ready')
  })
})
