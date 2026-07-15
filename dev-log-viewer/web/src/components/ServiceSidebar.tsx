import type { ServiceStatus } from '../types/api'

interface ServiceSidebarProps {
  services: ServiceStatus[]
  collapsed?: boolean
  selectedServiceID?: string
  onSelectService(serviceID: string): void
}

export function ServiceSidebar({ services, collapsed = false, selectedServiceID = '', onSelectService }: ServiceSidebarProps) {
  const groups = groupServices(services)
  return (
    <aside className={collapsed ? 'service-sidebar service-sidebar--collapsed' : 'service-sidebar'} aria-label="Service catalog">
      <section className="service-group">
        <button
          aria-pressed={selectedServiceID === ''}
          className={selectedServiceID === '' ? 'service-item service-item--all service-item--selected' : 'service-item service-item--all'}
          onClick={() => onSelectService('')}
          type="button"
        >
          <span className="state-dot state-dot--all" aria-hidden="true" />
          <span className="service-main">
            <span className="service-name">All Services</span>
            <span className="service-meta">combined stream</span>
          </span>
          <span className="log-chip">ALL</span>
        </button>
      </section>
      {groups.map(([group, items]) => (
        <section className="service-group" key={group}>
          <h2>{group}</h2>
          {items.map((service) => (
            <button
              aria-pressed={selectedServiceID === service.id}
              className={selectedServiceID === service.id ? 'service-item service-item--selected' : 'service-item'}
              key={service.id}
              onClick={() => onSelectService(service.id)}
              type="button"
            >
              <span className={`state-dot state-dot--${service.process_state}`} aria-hidden="true" />
              <span className="service-main">
                <span className="service-name">{service.name}</span>
                <span className="service-meta">:{service.port}</span>
              </span>
              <span className={`log-chip log-chip--${service.log_state}`}>{service.log_state}</span>
            </button>
          ))}
        </section>
      ))}
    </aside>
  )
}

function groupServices(services: ServiceStatus[]): Array<[string, ServiceStatus[]]> {
  return (['backend', 'gateway', 'frontend'] as const).map((group) => [
    group,
    services.filter((service) => service.group === group),
  ])
}
