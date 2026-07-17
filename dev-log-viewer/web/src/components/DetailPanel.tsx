import type { LogRecord } from '../types/api'

interface DetailPanelProps {
  record: LogRecord | undefined
}

export function DetailPanel({ record }: DetailPanelProps) {
  return (
    <aside className="detail-panel" aria-label="Log detail">
      <h2>Entry Detail</h2>
      {record ? (
        <>
          <dl className="detail-grid">
            <div>
              <dt>service</dt>
              <dd>{record.service}</dd>
            </div>
            <div>
              <dt>level</dt>
              <dd>{record.level}</dd>
            </div>
            {record.caller ? (
              <div>
                <dt>caller</dt>
                <dd>{record.caller}</dd>
              </div>
            ) : null}
            {record.request_id ? (
              <div>
                <dt>request_id</dt>
                <dd>{record.request_id}</dd>
              </div>
            ) : null}
            {record.trace_id ? (
              <div>
                <dt>trace_id</dt>
                <dd>{record.trace_id}</dd>
              </div>
            ) : null}
          </dl>
          <pre>{record.lines.join('\n')}</pre>
        </>
      ) : (
        <p className="muted">No entry selected</p>
      )}
    </aside>
  )
}
