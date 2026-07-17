import type { LogRecord, ServiceStatus } from '../types/api'

export const SERVICES: ServiceStatus[] = [
  service('identity-service', 'backend', 50061, 'running', 'ready'),
  service('recruitment-service', 'backend', 50062, 'running', 'ready'),
  service('interview-service', 'backend', 50063, 'offline', 'missing'),
  service('offer-service', 'backend', 50064, 'unknown', 'ready'),
  service('notification-service', 'backend', 50065, 'running', 'ready'),
  service('ai-agent-service', 'backend', 50066, 'running', 'ready'),
  service('analytics-service', 'backend', 50067, 'running', 'ready'),
  service('worker-service', 'backend', 50068, 'offline', 'missing'),
  service('smart-recruit-gateway', 'gateway', 8080, 'running', 'ready'),
  service('hr-frontend', 'frontend', 5173, 'running', 'ready'),
  service('user-frontend', 'frontend', 5174, 'running', 'ready'),
  service('interviewer-frontend', 'frontend', 5175, 'running', 'ready'),
]

export function createDemoRecords(count = 240): LogRecord[] {
  const levels: LogRecord['level'][] = ['INFO', 'DEBUG', 'WARN', 'ERROR']
  return Array.from({ length: count }, (_, index) => {
    const service = SERVICES[index % SERVICES.length]
    const level = levels[index % levels.length]
    const request = `req-${String(index % 17).padStart(3, '0')}`
    const message =
      level === 'ERROR'
        ? `failed to refresh service catalog attempt=${index % 5}`
        : `stream event processed offset=${index * 37} generation=${Math.floor(index / 32)}`
    return {
      id: index + 1,
      generation: Math.floor(index / 32),
      service: service.id,
      observed_at: new Date(Date.UTC(2026, 6, 15, 9, index % 60, index % 60)).toISOString(),
      source_time: new Date(Date.UTC(2026, 6, 15, 9, index % 60, index % 60)).toISOString(),
      level,
      caller: index % 3 === 0 ? 'internal/server/server.go:128' : 'internal/tailer/coordinator.go:55',
      message,
      lines: [message],
      request_id: request,
      trace_id: `trace-${String(index % 11).padStart(3, '0')}`,
      fields: {
        request_id: request,
        elapsed: `${(index % 9) + 1}ms`,
      },
    }
  })
}

function service(
  id: string,
  group: ServiceStatus['group'],
  port: number,
  process_state: ServiceStatus['process_state'],
  log_state: ServiceStatus['log_state'],
): ServiceStatus {
  return {
    id,
    name: id,
    group,
    port,
    process_state,
    log_state,
    log_size_bytes: log_state === 'ready' ? 4096 + port : 0,
    log_updated_at: log_state === 'ready' ? '2026-07-15T09:00:00Z' : undefined,
  }
}
