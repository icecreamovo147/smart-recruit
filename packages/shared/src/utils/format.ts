export const formatFileSize = (value: number): string => {
  if (!value) return '0 KB'
  if (value >= 1024 * 1024) return `${(value / 1024 / 1024).toFixed(1)} MB`
  return `${Math.ceil(value / 1024)} KB`
}

export const formatUploadedAt = (value: string): string => {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { hour12: false })
}

export const formatDateTime = (value: string): string => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (num: number): string => String(num).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

export const parseUnixTimestamp = (value: unknown): number | null => {
  if (value === null || value === undefined || value === '') return null
  const timestamp = typeof value === 'number' ? value : Number(value)
  if (!Number.isFinite(timestamp) || timestamp <= 0) return null
  return timestamp < 1_000_000_000_000 ? timestamp * 1000 : timestamp
}

export const formatUnixDateTime = (value: unknown, fallback = '-'): string => {
  const timestamp = parseUnixTimestamp(value)
  if (timestamp === null) return fallback
  const date = new Date(timestamp)
  return Number.isNaN(date.getTime()) ? fallback : date.toLocaleString('zh-CN', { hour12: false })
}
