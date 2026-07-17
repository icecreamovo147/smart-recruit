import type { Filters } from '../state/logState'
import type { LogLevel, ServiceStatus } from '../types/api'

interface FilterBarProps {
  filters: Filters
  services: readonly ServiceStatus[]
  onTextChange(value: string): void
  onCorrelationChange(value: string): void
  onLevelChange(value: LogLevel | ''): void
  onServiceChange(value: string): void
}

export function FilterBar({ filters, services, onTextChange, onCorrelationChange, onLevelChange, onServiceChange }: FilterBarProps) {
  const selectedLevel = Array.from(filters.levels)[0] ?? ''
  const selectedService = Array.from(filters.services)[0] ?? ''

  return (
    <section className="filter-bar" aria-label="Log filters">
      <label>
        <span>Search</span>
        <input value={filters.text} onChange={(event) => onTextChange(event.target.value)} placeholder="message text" />
      </label>
      <label>
        <span>Level</span>
        <select value={selectedLevel} onChange={(event) => onLevelChange(event.target.value as LogLevel | '')}>
          <option value="">All levels</option>
          <option value="ERROR">ERROR</option>
          <option value="FATAL">FATAL</option>
          <option value="PANIC">PANIC</option>
          <option value="DPANIC">DPANIC</option>
          <option value="WARN">WARN</option>
          <option value="INFO">INFO</option>
          <option value="DEBUG">DEBUG</option>
          <option value="UNKNOWN">UNKNOWN</option>
        </select>
      </label>
      <label>
        <span>Correlation</span>
        <input value={filters.correlation} onChange={(event) => onCorrelationChange(event.target.value)} placeholder="request_id / trace_id" />
      </label>
      <label>
        <span>Service</span>
        <select value={selectedService} onChange={(event) => onServiceChange(event.target.value)}>
          <option value="">All services</option>
          {services.map((service) => (
            <option key={service.id} value={service.id}>{service.name}</option>
          ))}
        </select>
      </label>
    </section>
  )
}
