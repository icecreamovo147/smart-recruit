import type { Component } from 'vue'

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
  icon?: Component
  danger?: boolean
  disabled?: boolean
  divided?: boolean
}
