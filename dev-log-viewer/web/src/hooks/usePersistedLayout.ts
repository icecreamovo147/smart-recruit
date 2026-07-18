import { useEffect, useState } from 'react'
import { DEFAULT_LAYOUT, loadLayout, saveLayout, type PersistedLayout } from '../utils/layoutStorage'

export function usePersistedLayout(storage: Storage | undefined = globalThis.localStorage): [PersistedLayout, (layout: PersistedLayout) => void] {
  const [layout, setLayoutState] = useState<PersistedLayout>(() => loadLayout(storage))

  useEffect(() => {
    saveLayout(storage, layout)
  }, [layout, storage])

  function setLayout(next: PersistedLayout): void {
    setLayoutState({
      sidebarCollapsed: next.sidebarCollapsed,
      detailWidth: next.detailWidth >= 280 && next.detailWidth <= 480 ? next.detailWidth : DEFAULT_LAYOUT.detailWidth,
    })
  }

  return [layout, setLayout]
}
