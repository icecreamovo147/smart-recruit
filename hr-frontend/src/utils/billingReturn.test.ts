import { describe, expect, it } from 'vitest'
import {
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
