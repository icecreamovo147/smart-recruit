import { useEffect, useMemo, useState } from 'react'
import { fetchServices } from '../api/services'
import { LogStreamController } from '../hooks/useLogStream'
import {
  applyEnvelope,
  clearRecords,
  createInitialState,
  markMalformedEvent,
  selectFilteredRecords,
  setConnectionStatus,
  setFilters,
  setFollowLatest,
  setRenderPaused,
  type LogViewerState,
} from '../state/logState'
import type { LogLevel } from '../types/api'
import { copyText } from '../utils/clipboard'
import { createExportBlob, downloadBlob, exportFileName } from '../utils/exportLogs'
import { usePersistedLayout } from '../hooks/usePersistedLayout'
import { DetailPanel } from './DetailPanel'
import { FilterBar } from './FilterBar'
import { LogTable } from './LogTable'
import { ServiceSidebar } from './ServiceSidebar'
import { StatusBar } from './StatusBar'
import { TopToolbar } from './TopToolbar'

export function AppShell() {
  const [viewerState, setViewerState] = useState<LogViewerState>(() => createInitialState())
  const [visibleRecords, setVisibleRecords] = useState(viewerState.records)
  const [selectedID, setSelectedID] = useState<number>(0)
  const [feedback, setFeedback] = useState('Sensitive local logs. Keep this viewer on loopback only.')
  const [layout, setLayout] = usePersistedLayout()
  const selectedServiceID = Array.from(viewerState.filters.services)[0] ?? ''
  const selectedRecord = visibleRecords.find((record) => record.id === selectedID) ?? visibleRecords[0]

  const streamController = useMemo(
    () => new LogStreamController({
      createEventSource: (url) => new EventSource(url),
      onEnvelope: (envelope) => setViewerState((current) => applyEnvelope(current, envelope)),
      onMalformed: () => setViewerState((current) => markMalformedEvent(current)),
      onStatus: (status) => setViewerState((current) => setConnectionStatus(current, status)),
    }),
    [],
  )

  useEffect(() => {
    let active = true
    fetchServices()
      .then((response) => {
        if (!active) return
        setViewerState((current) => ({ ...current, services: response.services }))
        setFeedback('Connected to local log stream.')
      })
      .catch(() => {
        if (!active) return
        setFeedback('Failed to load service catalog. Check .dev/logs/dev-log-viewer.log.')
      })
    return () => {
      active = false
    }
  }, [])

  useEffect(() => {
    if (viewerState.services.length === 0) return
    streamController.connect([], 300)
    return () => streamController.close()
  }, [streamController, viewerState.services.length])

  useEffect(() => {
    if (viewerState.renderPaused) return
    const next = selectVisibleRecords(viewerState)
    setVisibleRecords(next)
    if (next.length > 0 && !next.some((record) => record.id === selectedID)) {
      setSelectedID(next[0]?.id ?? 0)
    }
  }, [selectedID, viewerState])

  function togglePaused() {
    setViewerState((current) => {
      const nextPaused = !current.renderPaused
      const next = setRenderPaused(current, nextPaused)
      if (!nextPaused) setVisibleRecords(selectVisibleRecords(next))
      return next
    })
  }

  function jumpLatest() {
    setViewerState((current) => {
      const next = setFollowLatest(current, true)
      setVisibleRecords(selectVisibleRecords(next))
      return next
    })
    setFeedback('Jumped to latest buffered log.')
  }

  function clearView() {
    setViewerState((current) => clearRecords(current))
    setVisibleRecords([])
    setFeedback('Client buffer cleared. Log files were not modified.')
  }

  function exportLogs() {
    try {
      const blob = createExportBlob(visibleRecords)
      const fileName = exportFileName()
      downloadBlob(blob, fileName)
      setFeedback(`Exported ${fileName} (${blob.size} bytes, UTF-8).`)
    } catch {
      setFeedback('Export failed. Current view is unchanged.')
    }
  }

  async function copySelected() {
    if (!selectedRecord) {
      setFeedback('Copy skipped. No selected log row.')
      return
    }
    try {
      await copyText(selectedRecord.lines.join('\n'))
      setFeedback('Copied selected log lines.')
    } catch {
      setFeedback('Copy failed. Clipboard permission may be blocked.')
    }
  }

  function reconnectStream() {
    streamController.connect([], 300)
    setFeedback('Manual reconnect requested. Retrying the local stream.')
  }

  function updateFilters(nextFilters: Parameters<typeof setFilters>[1]) {
    setViewerState((current) => {
      const next = setFilters(current, nextFilters)
      if (!next.renderPaused) setVisibleRecords(selectVisibleRecords(next))
      return next
    })
  }

  function selectService(serviceID: string) {
    updateFilters({ services: serviceID ? new Set([serviceID]) : new Set() })
    setFeedback(serviceID ? `Showing logs for ${serviceID}.` : 'Showing logs for all services.')
  }

  return (
    <div className={layout.sidebarCollapsed ? 'app-shell app-shell--collapsed' : 'app-shell'}>
      <TopToolbar
        total={viewerState.records.length}
        paused={viewerState.renderPaused}
        unseen={viewerState.unseen}
        connectionStatus={viewerState.connectionStatus}
        onPause={togglePaused}
        onJumpLatest={jumpLatest}
        onClear={clearView}
        onCopy={copySelected}
        onExport={exportLogs}
        onReconnect={reconnectStream}
        onToggleSidebar={() => setLayout({ ...layout, sidebarCollapsed: !layout.sidebarCollapsed })}
      />
      <div className="state-banner" role="status">
        <span>{feedback}</span>
        <div className="state-banner__tokens">
          <strong>{viewerState.connectionStatus === 'reconnecting' ? 'Reconnecting stream' : 'Loopback + sensitive logs'}</strong>
          {viewerState.renderPaused ? <strong>Paused: incoming logs continue buffering.</strong> : null}
          {viewerState.dropped > 0 ? <strong>High volume: {viewerState.dropped.toLocaleString()} rows dropped or evicted.</strong> : null}
          {viewerState.resets > 0 ? <strong>Reset marker visible.</strong> : null}
          {viewerState.services.length === 0 ? <strong>No services configured.</strong> : null}
          {visibleRecords.length === 0 ? <strong>No results.</strong> : null}
        </div>
      </div>
      <div className="app-body" style={{ gridTemplateColumns: layout.sidebarCollapsed ? `var(--sidebar-width-collapsed) minmax(420px, 1fr) ${layout.detailWidth}px` : `var(--sidebar-width-expanded) minmax(420px, 1fr) ${layout.detailWidth}px` }}>
        <ServiceSidebar
          services={viewerState.services}
          collapsed={layout.sidebarCollapsed}
          selectedServiceID={selectedServiceID}
          onSelectService={selectService}
        />
        <main className="log-workspace" aria-label="Live log workspace">
          <FilterBar
            filters={viewerState.filters}
            services={viewerState.services}
            onTextChange={(text) => updateFilters({ text })}
            onCorrelationChange={(correlation) => updateFilters({ correlation })}
            onLevelChange={(level: LogLevel | '') => updateFilters({ levels: level ? new Set([level]) : new Set() })}
            onServiceChange={(service) => updateFilters({ services: service ? new Set([service]) : new Set() })}
          />
          <LogTable records={visibleRecords} selectedID={selectedID} onSelect={setSelectedID} />
        </main>
        <DetailPanel record={selectedRecord} />
      </div>
      <StatusBar services={viewerState.services} total={viewerState.records.length} />
    </div>
  )
}

function selectVisibleRecords(state: LogViewerState) {
  return selectFilteredRecords(state).slice().reverse()
}
