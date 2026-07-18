import type { LogRecord } from '../types/api'

export function formatLogExport(records: readonly LogRecord[]): string {
  return records
    .map((record) => {
      const time = record.source_time ?? record.observed_at
      return `[${time}] ${record.level} ${record.service} ${record.caller ?? ''} ${record.lines.join('\n')}`.trim()
    })
    .join('\n')
}

export function createExportBlob(records: readonly LogRecord[]): Blob {
  return new Blob([formatLogExport(records)], { type: 'text/plain;charset=utf-8' })
}

export function exportFileName(now = new Date()): string {
  return `dev-log-viewer-${now.toISOString().replace(/[:.]/g, '-')}.log`
}

export interface DownloadTarget {
  createObjectURL(blob: Blob): string
  revokeObjectURL(url: string): void
}

export interface DownloadDocument {
  createElement(tagName: 'a'): HTMLAnchorElement
  body: {
    appendChild(node: Node): Node
    removeChild(node: Node): Node
  }
}

export function downloadBlob(blob: Blob, fileName: string, target: DownloadTarget = URL, documentRef: DownloadDocument = document): void {
  const url = target.createObjectURL(blob)
  const anchor = documentRef.createElement('a')
  anchor.href = url
  anchor.download = fileName
  anchor.rel = 'noopener'
  documentRef.body.appendChild(anchor)
  anchor.click()
  documentRef.body.removeChild(anchor)
  target.revokeObjectURL(url)
}
