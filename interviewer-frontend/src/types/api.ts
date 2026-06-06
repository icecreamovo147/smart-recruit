export class BusinessError extends Error {
  code: number
  requestId: string

  constructor(code: number, message: string, requestId = '') {
    super(message)
    this.name = 'BusinessError'
    this.code = code
    this.requestId = requestId
  }
}
