import { catalogs, type MessageKey } from './generated'

export type AppLocale = keyof typeof catalogs
export type MessageArgs = Record<string, string | number | boolean>

export const DEFAULT_LOCALE: AppLocale = 'zh-CN'

let currentLocale: AppLocale = DEFAULT_LOCALE

export function isAppLocale(value: unknown): value is AppLocale {
  return value === 'zh-CN' || value === 'en-US'
}

export function configureLocale(value: unknown): AppLocale {
  currentLocale = isAppLocale(value) ? value : DEFAULT_LOCALE
  if (typeof document !== 'undefined') {
    document.documentElement.lang = currentLocale
  }
  return currentLocale
}

export function getLocale(): AppLocale {
  return currentLocale
}

export function t(key: MessageKey, args: MessageArgs = {}): string {
  const message = catalogs[currentLocale][key] || catalogs[currentLocale]['common.unknown_error']
  return message.replace(/\{([a-zA-Z][a-zA-Z0-9_]*)\}/g, (token, name: string) => (
    Object.prototype.hasOwnProperty.call(args, name) ? String(args[name]) : token
  ))
}

interface RuntimeConfigEnvelope {
  data?: {
    locale?: unknown
  }
}

export async function initializeLocale(apiBaseURL = ''): Promise<AppLocale> {
  try {
    const response = await fetch(`${apiBaseURL}/api/v1/public/runtime-config`, {
      credentials: 'include',
      headers: { 'X-Client-App': 'runtime-config' },
    })
    if (!response.ok) return configureLocale(DEFAULT_LOCALE)
    const payload = await response.json() as RuntimeConfigEnvelope
    return configureLocale(payload.data?.locale)
  } catch {
    return configureLocale(DEFAULT_LOCALE)
  }
}

export function localizedBackendMessage(
  payload: { message_key?: string; msg?: string } | undefined,
  fallback: MessageKey = 'common.operation_failed',
): string {
  if (payload?.message_key && payload.msg) return payload.msg
  return t(fallback)
}

export type { MessageKey }
