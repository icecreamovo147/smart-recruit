import type { AxiosError } from 'axios'

export interface ApiResponse<T = unknown> {
  code: number
  msg: string
  data: T
  request_id?: string
}

export class BusinessError extends Error {
  code: number
  requestId?: string

  constructor(code: number, message: string, requestId?: string) {
    super(message)
    this.name = 'BusinessError'
    this.code = code
    this.requestId = requestId
  }
}

export type NetworkError = AxiosError
