import { describe, expect, it } from 'vitest'
import { copyText } from '../utils/clipboard'
import { downloadBlob, formatLogExport } from '../utils/exportLogs'
import { DEFAULT_LAYOUT, LAYOUT_STORAGE_KEY, loadLayout, saveLayout, type StorageLike } from '../utils/layoutStorage'

describe('export and persisted layout utilities', () => {
  it('exports current records as utf-8 log text in order', () => {
    const text = formatLogExport([
      {
        id: 1,
        generation: 0,
        service: 'svc',
        observed_at: '2026-07-15T09:00:00Z',
        level: 'INFO',
        message: 'hello',
        lines: ['hello', 'world'],
      },
    ])
    expect(text).toContain('INFO svc')
    expect(text).toContain('hello\nworld')
  })

  it('loads only valid layout values and ignores invalid writes', () => {
    const storage = new MemoryStorage()
    storage.setItem(LAYOUT_STORAGE_KEY, JSON.stringify({ sidebarCollapsed: true, detailWidth: 420 }))
    expect(loadLayout(storage)).toEqual({ sidebarCollapsed: true, detailWidth: 420 })

    storage.setItem(LAYOUT_STORAGE_KEY, JSON.stringify({ sidebarCollapsed: 'yes', detailWidth: 999 }))
    expect(loadLayout(storage)).toEqual(DEFAULT_LAYOUT)

    saveLayout(storage, { sidebarCollapsed: true, detailWidth: 320 })
    expect(loadLayout(storage)).toEqual({ sidebarCollapsed: true, detailWidth: 320 })
    saveLayout(storage, { sidebarCollapsed: true, detailWidth: 900 })
    expect(loadLayout(storage)).toEqual({ sidebarCollapsed: true, detailWidth: 320 })
  })

  it('surfaces clipboard failures', async () => {
    await expect(copyText('x', undefined)).rejects.toThrow('Clipboard API is unavailable')
  })

  it('downloads generated blobs through a revocable object url', () => {
    const calls: string[] = []
    const anchor = {
      href: '',
      download: '',
      rel: '',
      click: () => calls.push('click'),
    } as unknown as HTMLAnchorElement

    downloadBlob(
      new Blob(['hello']),
      'current.log',
      {
        createObjectURL: () => {
          calls.push('create')
          return 'blob:local'
        },
        revokeObjectURL: () => calls.push('revoke'),
      },
      {
        createElement: () => anchor,
        body: {
          appendChild: () => {
            calls.push('append')
            return anchor
          },
          removeChild: () => {
            calls.push('remove')
            return anchor
          },
        },
      },
    )

    expect(anchor.download).toBe('current.log')
    expect(calls).toEqual(['create', 'append', 'click', 'remove', 'revoke'])
  })
})

class MemoryStorage implements StorageLike {
  private values = new Map<string, string>()

  getItem(key: string): string | null {
    return this.values.get(key) ?? null
  }

  setItem(key: string, value: string): void {
    this.values.set(key, value)
  }
}
