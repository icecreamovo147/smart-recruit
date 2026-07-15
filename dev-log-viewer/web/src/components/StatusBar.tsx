import type { ServiceStatus } from '../types/api'

interface StatusBarProps {
  services: ServiceStatus[]
  total: number
}

export function StatusBar({ services, total }: StatusBarProps) {
  const running = services.filter((service) => service.process_state === 'running').length
  return (
    <footer className="status-bar">
      <span>{running}/{services.length} processes running</span>
      <span>{total.toLocaleString()} rows in client buffer</span>
      <span>Loopback only</span>
    </footer>
  )
}
