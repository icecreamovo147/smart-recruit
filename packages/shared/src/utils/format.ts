export const BUSINESS_TIME_ZONE = 'Asia/Shanghai'
export const BUSINESS_UTC_OFFSET = '+08:00'

const absoluteTimePattern = /(Z|[+-]\d{2}:?\d{2})$/i
const wallClockPattern = /^(\d{4}-\d{2}-\d{2})[ T](\d{2}:\d{2})(?::(\d{2})(\.\d{1,9})?)?$/

export const parseBusinessDateTime = (value: string | number | Date): Date => {
  if (value instanceof Date || typeof value === 'number') return new Date(value)
  const source = value.trim()
  if (!source || /^\d{4}-\d{2}-\d{2}$/.test(source) || absoluteTimePattern.test(source)) return new Date(source)
  const match = source.match(wallClockPattern)
  if (!match) return new Date(source)
  return new Date(`${match[1]}T${match[2]}:${match[3] || '00'}${match[4] || ''}${BUSINESS_UTC_OFFSET}`)
}

const partsFormatter = (withSeconds: boolean) => new Intl.DateTimeFormat('zh-CN', {
  timeZone: BUSINESS_TIME_ZONE,
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  ...(withSeconds ? { second: '2-digit' as const } : {}),
  hourCycle: 'h23',
})

const shanghaiParts = (date: Date, withSeconds: boolean): Record<string, string> =>
  Object.fromEntries(partsFormatter(withSeconds).formatToParts(date).map((part) => [part.type, part.value]))

export const formatShanghaiDateTime = (
  value: string | number | Date | null | undefined,
  fallback = '-',
  withSeconds = true,
): string => {
  if (value === null || value === undefined || value === '') return fallback
  const date = parseBusinessDateTime(value)
  if (Number.isNaN(date.getTime())) return typeof value === 'string' ? value : fallback
  const parts = shanghaiParts(date, withSeconds)
  const seconds = withSeconds ? `:${parts.second}` : ''
  return `${parts.year}-${parts.month}-${parts.day} ${parts.hour}:${parts.minute}${seconds}`
}

export const formatShanghaiTime = (
  value: string | number | Date | null | undefined,
  fallback = '',
): string => {
  const formatted = formatShanghaiDateTime(value, fallback, false)
  return formatted === fallback ? fallback : formatted.slice(11, 16)
}

export const formatShanghaiLongDate = (
  value: string | number | Date | null | undefined,
  fallback = '-',
): string => {
  if (value === null || value === undefined || value === '') return fallback
  const date = parseBusinessDateTime(value)
  if (Number.isNaN(date.getTime())) return fallback
  return new Intl.DateTimeFormat('zh-CN', {
    timeZone: BUSINESS_TIME_ZONE,
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    weekday: 'long',
  }).format(date)
}

// Serialize a datetime-local wall clock or an absolute Date/string to the API
// contract's RFC3339 representation in Asia/Shanghai.
export const toShanghaiRFC3339 = (value: string | number | Date): string => {
  if (typeof value === 'string' && !absoluteTimePattern.test(value.trim())) {
    const match = value.trim().match(wallClockPattern)
    if (match) return `${match[1]}T${match[2]}:${match[3] || '00'}${BUSINESS_UTC_OFFSET}`
  }
  const date = parseBusinessDateTime(value)
  if (Number.isNaN(date.getTime())) throw new Error('无效的日期时间')
  const parts = shanghaiParts(date, true)
  return `${parts.year}-${parts.month}-${parts.day}T${parts.hour}:${parts.minute}:${parts.second}${BUSINESS_UTC_OFFSET}`
}

// Plain dates (birthdays, start dates) are calendar values, not instants.
export const formatPlainDate = (value: string | null | undefined, fallback = '-'): string => {
  if (!value) return fallback
  const match = value.match(/^(\d{4})-(\d{2})-(\d{2})/)
  return match ? `${match[1]}-${match[2]}-${match[3]}` : value
}

export const formatFileSize = (value: number): string => {
  if (!value) return '0 KB'
  if (value >= 1024 * 1024) return `${(value / 1024 / 1024).toFixed(1)} MB`
  return `${Math.ceil(value / 1024)} KB`
}

export const formatUploadedAt = (value: string): string => {
  return formatShanghaiDateTime(value, '')
}

export const formatDateTime = (value: string): string => {
  return formatShanghaiDateTime(value, '-', false)
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
  return formatShanghaiDateTime(timestamp, fallback)
}
