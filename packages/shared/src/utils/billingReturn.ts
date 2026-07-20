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
  'payment_return',
] as const

export const isAlipayReturnSearch = (search: string): boolean => {
  const params = new URLSearchParams(search.startsWith('?') ? search.slice(1) : search)
  const method = params.get('method') || ''
  return Boolean(params.get('payment_return') || (params.get('out_trade_no') && params.get('trade_no') && (method.includes('return') || params.get('sign'))))
}

export const getPaymentReturnToken = (search: string): string => {
	const params = new URLSearchParams(search.startsWith('?') ? search.slice(1) : search)
	return params.get('payment_return') || ''
}

export const detectAlipayScene = (navigatorValue: Pick<Navigator, 'userAgent'> & { userAgentData?: { mobile?: boolean } } = navigator): 'desktop' | 'wap' => {
	if (navigatorValue.userAgentData?.mobile === true) return 'wap'
	return /Android|iPhone|iPad|iPod|Mobile/i.test(navigatorValue.userAgent) ? 'wap' : 'desktop'
}

const ALIPAY_GATEWAY_HOSTS = new Set([
  'openapi-sandbox.dl.alipaydev.com',
  'openapi.alipaydev.com',
  'openapi.alipay.com',
])

export interface AlipayPaymentForm {
  action: string
  fields: Record<'biz_content', string>
}

/**
 * Alipay page/wap pay is a page-execute API: public parameters stay in the
 * gateway query string while biz_content is submitted as a POST form field.
 * Navigating to the signed URL with GET can return an empty sandbox response.
 */
export const buildAlipayPaymentForm = (redirectUrl: string): AlipayPaymentForm => {
  const gateway = new URL(redirectUrl)
  const method = gateway.searchParams.get('method') || ''
  const bizContent = gateway.searchParams.get('biz_content') || ''

  if (gateway.protocol !== 'https:' || !ALIPAY_GATEWAY_HOSTS.has(gateway.hostname) || gateway.pathname !== '/gateway.do') {
    throw new Error('支付宝收银台地址不受信任')
  }
  if (method !== 'alipay.trade.page.pay' && method !== 'alipay.trade.wap.pay') {
    throw new Error('支付宝收银台接口不受支持')
  }
  if (!bizContent || gateway.searchParams.get('charset')?.toLowerCase() !== 'utf-8' || gateway.searchParams.get('sign_type') !== 'RSA2' || !gateway.searchParams.get('sign')) {
    throw new Error('支付宝收银台参数不完整')
  }

  gateway.searchParams.delete('biz_content')
  gateway.hash = ''
  return { action: gateway.toString(), fields: { biz_content: bizContent } }
}

export const submitAlipayPayment = (redirectUrl: string): void => {
  const payment = buildAlipayPaymentForm(redirectUrl)
  const form = document.createElement('form')
  form.method = 'POST'
  form.action = payment.action
  form.acceptCharset = 'UTF-8'
  form.style.display = 'none'

  for (const [name, value] of Object.entries(payment.fields)) {
    const input = document.createElement('input')
    input.type = 'hidden'
    input.name = name
    input.value = value
    form.appendChild(input)
  }

  document.body.appendChild(form)
  form.submit()
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
