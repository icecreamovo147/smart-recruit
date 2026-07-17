import type { ServicesResponse } from '../types/api'

export async function fetchServices(fetcher: typeof fetch = fetch): Promise<ServicesResponse> {
  const response = await fetcher('/api/v1/services', {
    method: 'GET',
    headers: { Accept: 'application/json' },
  })
  if (!response.ok) {
    throw new Error(`Failed to load services: ${response.status}`)
  }
  return decodeServicesResponse(await response.json())
}

export function decodeServicesResponse(value: unknown): ServicesResponse {
  if (!isRecord(value) || !Array.isArray(value.services)) {
    throw new Error('Malformed services response')
  }
  return {
    services: value.services.map((service) => {
      if (!isRecord(service) || typeof service.id !== 'string' || typeof service.name !== 'string') {
        throw new Error('Malformed service entry')
      }
      return {
        id: service.id,
        name: service.name,
        group: assertString(service.group) as ServicesResponse['services'][number]['group'],
        port: assertNumber(service.port),
        process_state: assertString(service.process_state) as ServicesResponse['services'][number]['process_state'],
        log_state: assertString(service.log_state) as ServicesResponse['services'][number]['log_state'],
        log_size_bytes: assertNumber(service.log_size_bytes),
        log_updated_at: typeof service.log_updated_at === 'string' ? service.log_updated_at : undefined,
      }
    }),
  }
}

function assertString(value: unknown): string {
  if (typeof value !== 'string') throw new Error('Expected string')
  return value
}

function assertNumber(value: unknown): number {
  if (typeof value !== 'number') throw new Error('Expected number')
  return value
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
