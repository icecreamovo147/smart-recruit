export const LAYOUT_STORAGE_KEY = 'dev-log-viewer:layout:v1'

export interface PersistedLayout {
  sidebarCollapsed: boolean
  detailWidth: number
}

export const DEFAULT_LAYOUT: PersistedLayout = {
  sidebarCollapsed: false,
  detailWidth: 336,
}

export interface StorageLike {
  getItem(key: string): string | null
  setItem(key: string, value: string): void
}

export function loadLayout(storage: StorageLike | undefined): PersistedLayout {
  if (!storage) return DEFAULT_LAYOUT
  try {
    const raw = storage.getItem(LAYOUT_STORAGE_KEY)
    if (!raw) return DEFAULT_LAYOUT
    const parsed: unknown = JSON.parse(raw)
    if (!isRecord(parsed)) return DEFAULT_LAYOUT
    const sidebarCollapsed = typeof parsed.sidebarCollapsed === 'boolean' ? parsed.sidebarCollapsed : DEFAULT_LAYOUT.sidebarCollapsed
    const detailWidth = typeof parsed.detailWidth === 'number' && parsed.detailWidth >= 280 && parsed.detailWidth <= 480
      ? parsed.detailWidth
      : DEFAULT_LAYOUT.detailWidth
    return { sidebarCollapsed, detailWidth }
  } catch {
    return DEFAULT_LAYOUT
  }
}

export function saveLayout(storage: StorageLike | undefined, layout: PersistedLayout): void {
  if (!storage) return
  if (layout.detailWidth < 280 || layout.detailWidth > 480) return
  try {
    storage.setItem(LAYOUT_STORAGE_KEY, JSON.stringify(layout))
  } catch {
    // localStorage can be unavailable in private or restricted contexts.
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
