import { describe, expect, it } from 'vitest'
import { LOG_ROW_HEIGHT } from '../components/LogTable'
import { SERVICES, createDemoRecords } from '../components/demoData'

describe('canonical UI data contract', () => {
  it('uses the SPEC service catalog facts', () => {
    expect(SERVICES.map((service) => [service.id, service.group, service.port])).toEqual([
      ['identity-service', 'backend', 50061],
      ['recruitment-service', 'backend', 50062],
      ['interview-service', 'backend', 50063],
      ['offer-service', 'backend', 50064],
      ['notification-service', 'backend', 50065],
      ['ai-agent-service', 'backend', 50066],
      ['analytics-service', 'backend', 50067],
      ['worker-service', 'backend', 50068],
      ['smart-recruit-gateway', 'gateway', 8080],
      ['hr-frontend', 'frontend', 5173],
      ['user-frontend', 'frontend', 5174],
      ['platform-frontend', 'frontend', 5175],
    ])
  })

  it('keeps demo logs large enough for virtualization without rendering all rows', () => {
    expect(createDemoRecords(10_000)).toHaveLength(10_000)
    expect(LOG_ROW_HEIGHT).toBe(28)
  })
})
