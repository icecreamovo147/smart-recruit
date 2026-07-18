interface TopToolbarProps {
  total: number
  paused: boolean
  unseen: number
  connectionStatus: string
  onPause(): void
  onJumpLatest(): void
  onClear(): void
  onCopy(): void
  onExport(): void
  onReconnect(): void
  onToggleSidebar(): void
}

export function TopToolbar({ total, paused, unseen, connectionStatus, onPause, onJumpLatest, onClear, onCopy, onExport, onReconnect, onToggleSidebar }: TopToolbarProps) {
  const statusClass = connectionStatus === 'connected' ? 'status-token status-token--ok' : 'status-token status-token--warn'
  return (
    <header className="top-toolbar">
      <div>
        <p className="eyebrow">Local Diagnostics</p>
        <h1>Dev Log Viewer</h1>
      </div>
      <div className="toolbar-actions" aria-label="Stream status">
        <span className={statusClass}>{connectionStatus}</span>
        <button type="button" onClick={onToggleSidebar}>Sidebar</button>
        <button type="button" onClick={onPause}>{paused ? 'Resume' : 'Pause'}</button>
        <button type="button" onClick={onJumpLatest}>Latest {unseen > 0 ? `(${unseen})` : ''}</button>
        <button type="button" onClick={onClear}>Clear</button>
        <button type="button" onClick={onCopy}>Copy</button>
        <button type="button" onClick={onExport}>Export</button>
        <button type="button" onClick={onReconnect}>Reconnect</button>
        <span className="metric-token">{total.toLocaleString()} buffered</span>
      </div>
    </header>
  )
}
