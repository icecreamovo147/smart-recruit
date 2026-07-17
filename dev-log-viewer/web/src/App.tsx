import { AppShell } from './components/AppShell'

export const APP_TITLE = 'Dev Log Viewer'

export function getPlaceholderStatus(): string {
  return 'Canonical UI ready'
}

export default function App() {
  return <AppShell />
}
