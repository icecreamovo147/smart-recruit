export type ConsoleStatus =
  | 'enabled'
  | 'disabled'
  | 'default'
  | 'bound'
  | 'unbound'
  | 'healthy'
  | 'error'
  | 'untested'
  | 'current'
  | 'warning'

export interface ConsoleAction {
  key: string
  label: string
  icon?: object
  danger?: boolean
  disabled?: boolean
  divided?: boolean
}
