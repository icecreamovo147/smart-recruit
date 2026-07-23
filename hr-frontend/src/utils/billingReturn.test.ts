import { describe, expect, it } from 'vitest'
import {
  buildAlipayPaymentForm,
  detectAlipayScene,
  getPaymentReturnToken,
  isAlipayReturnSearch,
  isBillingAlreadyPaidMessage,
  stripAlipayReturnQuery,
} from '@shared/utils/billingReturn'

describe('billingReturn helpers', () => {
  it('detects Alipay sync return query', () => {
    expect(isAlipayReturnSearch(
      '?charset=utf-8&out_trade_no=P1&method=alipay.trade.page.pay.return&trade_no=2026&sign=abc',
    )).toBe(true)
    expect(isAlipayReturnSearch('?foo=1')).toBe(false)
    expect(isAlipayReturnSearch('?payment_return=opaque-token')).toBe(true)
    expect(getPaymentReturnToken('?payment_return=opaque-token')).toBe('opaque-token')
  })

  it('selects WAP from device capability rather than viewport width', () => {
    expect(detectAlipayScene({ userAgent: 'Desktop', userAgentData: { mobile: false } })).toBe('desktop')
    expect(detectAlipayScene({ userAgent: 'Desktop', userAgentData: { mobile: true } })).toBe('wap')
    expect(detectAlipayScene({ userAgent: 'Mozilla/5.0 (iPhone)' })).toBe('wap')
  })

  it('builds the Alipay page-execute POST form without biz_content in the action URL', () => {
    const payment = buildAlipayPaymentForm(
      'https://openapi-sandbox.dl.alipaydev.com/gateway.do?app_id=9021&biz_content=%7B%22total_amount%22%3A%229.90%22%7D&charset=utf-8&method=alipay.trade.page.pay&sign=abc%2B123&sign_type=RSA2',
    )

    const action = new URL(payment.action)
    expect(action.hostname).toBe('openapi-sandbox.dl.alipaydev.com')
    expect(action.searchParams.has('biz_content')).toBe(false)
    expect(action.searchParams.get('sign')).toBe('abc+123')
    expect(payment.fields.biz_content).toBe('{"total_amount":"9.90"}')
  })

  it('rejects untrusted or incomplete Alipay payment URLs', () => {
    expect(() => buildAlipayPaymentForm(
      'https://example.com/gateway.do?biz_content=%7B%7D&charset=utf-8&method=alipay.trade.page.pay&sign=x&sign_type=RSA2',
    )).toThrow('地址不受信任')
    expect(() => buildAlipayPaymentForm(
      'https://openapi-sandbox.dl.alipaydev.com/gateway.do?charset=utf-8&method=alipay.trade.page.pay&sign=x&sign_type=RSA2',
    )).toThrow('参数不完整')
  })

  it('strips Alipay return params while keeping other query values', () => {
    expect(stripAlipayReturnQuery(
      'http://localhost:5173/hr/billing?out_trade_no=P1&trade_no=T1&sign=x&tab=orders',
    )).toBe('/hr/billing?tab=orders')
  })

  it('recognizes already-paid business messages', () => {
    expect(isBillingAlreadyPaidMessage('订单已经支付成功，请刷新套餐与订单状态')).toBe(true)
    expect(isBillingAlreadyPaidMessage('支付结果确认中，请稍候')).toBe(false)
  })
})
