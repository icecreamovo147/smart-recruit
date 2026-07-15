import { useVirtualizer } from '@tanstack/react-virtual'
import { useRef } from 'react'
import type { LogRecord } from '../types/api'

export const LOG_ROW_HEIGHT = 28

interface LogTableProps {
  records: LogRecord[]
  selectedID: number
  onSelect(id: number): void
}

export function LogTable({ records, selectedID, onSelect }: LogTableProps) {
  const parentRef = useRef<HTMLDivElement | null>(null)
  const rowVirtualizer = useVirtualizer({
    count: records.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => LOG_ROW_HEIGHT,
    overscan: 8,
  })

  return (
    <section className="log-panel" aria-label="Log entries">
      <div className="log-header" role="row">
        <span>Time</span>
        <span>Level</span>
        <span>Service</span>
        <span>Message</span>
      </div>
      <div className="log-scroll" ref={parentRef}>
        {records.length === 0 ? (
          <div className="empty-state">No results in the current client buffer.</div>
        ) : (
          <div className="log-virtual-space" style={{ height: `${rowVirtualizer.getTotalSize()}px` }}>
            {rowVirtualizer.getVirtualItems().map((virtualRow) => {
            const record = records[virtualRow.index]
            if (!record) return null
            return (
              <button
                className={record.id === selectedID ? 'log-row log-row--selected' : 'log-row'}
                key={record.id}
                onClick={() => onSelect(record.id)}
                style={{ transform: `translateY(${virtualRow.start}px)` }}
                type="button"
              >
                <time>{formatTime(record.observed_at)}</time>
                <span className={`level-badge level-badge--${record.level.toLowerCase()}`}>{record.level}</span>
                <span className="log-service">{record.service}</span>
                <span className="log-message">{record.message}</span>
              </button>
            )
            })}
          </div>
        )}
      </div>
    </section>
  )
}

function formatTime(value: string): string {
  return value.slice(11, 19)
}
