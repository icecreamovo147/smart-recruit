/** sessionStorage key for the order redirected to Alipay sandbox cashier */
export const BILLING_PENDING_ORDER_KEY = 'smart-recruit:billing-pending-order'

const ALIPAY_RETURN_QUERY_KEYS = [
  'charset',
  'out_trade_no',
  'method',
  'total_amount',
  'sign',
  'trade_no',
  'auth_app_id',
  'version',
  'app_id',
  'sign_type',
  'seller_id',
  'timestamp',
] as const

export const isAlipayReturnSearch = (search: string): boolean => {
  const params = new URLSearchParams(search.startsWith('?') ? search.slice(1) : search)
  const method = params.get('method') || ''
  return Boolean(params.get('out_trade_no') && params.get('trade_no') && (method.includes('return') || params.get('sign')))
}

export const stripAlipayReturnQuery = (href = typeof window !== 'undefined' ? window.location.href : ''): string => {
  const url = new URL(href, 'http://localhost')
  for (const key of ALIPAY_RETURN_QUERY_KEYS) {
    url.searchParams.delete(key)
  }
  const query = url.searchParams.toString()
  return `${url.pathname}${query ? `?${query}` : ''}${url.hash}`
}

export const rememberBillingPendingOrder = (orderNo: string): void => {
  if (typeof sessionStorage === 'undefined' || !orderNo) return
  sessionStorage.setItem(BILLING_PENDING_ORDER_KEY, orderNo)
}

export const takeBillingPendingOrder = (): string => {
  if (typeof sessionStorage === 'undefined') return ''
  const orderNo = sessionStorage.getItem(BILLING_PENDING_ORDER_KEY) || ''
  sessionStorage.removeItem(BILLING_PENDING_ORDER_KEY)
  return orderNo
}

export const isBillingAlreadyPaidMessage = (message: string): boolean => (
  message.includes('已经支付成功') || message.includes('支付成功')
)

export const sleep = (ms: number): Promise<void> => new Promise((resolve) => {
  window.setTimeout(resolve, ms)
})
