<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { createBillingOrder, getBillingAccount, getBillingCatalog, listBillingOrders, payBillingOrder, refundBillingOrder, type BillingAccount, type BillingOrder, type BillingProduct } from '@/api/billing'

const loading = ref(false)
const account = ref<BillingAccount | null>(null)
const products = ref<BillingProduct[]>([])
const orders = ref<BillingOrder[]>([])
const environment = ref('sandbox')

const load = async () => {
  loading.value = true
  try {
    const [catalog, current, history] = await Promise.all([getBillingCatalog(), getBillingAccount(), listBillingOrders()])
    products.value = catalog.products || []; environment.value = catalog.payment_environment || 'sandbox'
    account.value = current; orders.value = history.orders || []
  } finally { loading.value = false }
}

const purchase = async (product: BillingProduct, priceId: number, amountFen: number) => {
  if (amountFen <= 0) return
  await ElMessageBox.confirm(`确认购买「${product.name}」，沙箱标价 ¥${(amountFen / 100).toFixed(2)}；升级订单将由服务端按剩余周期计算差价。`, '支付宝沙箱支付', { type: 'warning' })
  const current = account.value?.subscription
  const orderType = product.product_type === 'credit_pack' ? 'credit_pack' : (!current ? 'subscribe' : current.product_key === product.product_key ? 'renew' : 'upgrade')
  const created = await createBillingOrder(priceId, orderType)
  const scene = window.matchMedia('(max-width: 768px)').matches ? 'wap' : 'desktop'
  const payment = await payBillingOrder(created.order.order_no, scene)
  if (payment.payment_environment !== 'sandbox') throw new Error('当前开发版本只允许支付宝沙箱支付')
  window.location.assign(payment.redirect_url)
}

const refund = async (order: BillingOrder) => {
  const { value } = await ElMessageBox.prompt('请填写退款原因；已消费额度的订单会转人工审核。', '申请退款', { inputPattern: /\S+/, inputErrorMessage: '退款原因不能为空' })
  await refundBillingOrder(order.order_no, value)
  ElMessage.success('退款申请已提交'); await load()
}

const formatTime = (value?: number) => value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'
onMounted(load)
</script>

<template>
  <section class="billing-page" v-loading="loading">
    <el-alert v-if="environment === 'sandbox'" title="当前为支付宝沙箱环境：所有金额和交易均为测试数据，不会产生真实扣款，也不计入真实营收。" type="warning" :closable="false" show-icon />
    <header class="billing-hero"><div><span>AI BILLING</span><h1>AI 套餐与额度</h1><p>按月获得额度，也可以购买十二个月有效的加量包。</p></div><div class="credit-card"><small>可用 AI 额度</small><strong>{{ (account?.available_credits || 0).toLocaleString() }}</strong><span>已预占 {{ account?.reserved_credits || 0 }}</span></div></header>
    <div v-if="account?.subscription" class="current-plan"><div><small>当前套餐</small><strong>{{ account.subscription.product_name }}</strong></div><div><small>有效期至</small><strong>{{ formatTime(account.subscription.current_period_end_unix_ms) }}</strong></div></div>
    <div class="product-grid"><article v-for="product in products" :key="product.id" class="product-card"><h2>{{ product.name }}</h2><p>{{ product.description }}</p><div v-for="price in product.prices" :key="price.id" class="price-row"><div><strong>{{ price.amount_fen ? `¥${(price.amount_fen / 100).toFixed(2)}` : '免费' }}</strong><span>/ {{ price.billing_term === 'yearly' ? '年' : price.billing_term === 'monthly' ? '月' : '次' }}</span><small>含 {{ price.included_credits.toLocaleString() }} AI 额度</small></div><el-button v-if="price.amount_fen > 0" type="primary" @click="purchase(product, price.id, price.amount_fen)">支付宝沙箱购买</el-button></div></article></div>
    <section class="orders"><h2>订单记录</h2><el-empty v-if="!orders.length" description="暂无订单" /><div v-for="order in orders" :key="order.order_no" class="order-row"><div><strong>{{ order.product_name }}</strong><span>{{ order.order_no }} · {{ formatTime(order.created_at_unix_ms) }}</span></div><div><el-tag type="warning">沙箱</el-tag><strong>¥{{ (order.amount_fen / 100).toFixed(2) }}</strong><el-tag>{{ order.status }}</el-tag><el-button v-if="order.status === 'paid'" link type="danger" @click="refund(order)">申请退款</el-button></div></div></section>
  </section>
</template>

<style scoped>
.billing-page{display:grid;gap:20px;padding:24px 0 48px}.billing-hero{display:flex;justify-content:space-between;gap:24px;padding:32px;border-radius:20px;background:linear-gradient(135deg,#14213d,#2457d6);color:#fff}.billing-hero h1{margin:6px 0;font-size:32px}.billing-hero p{margin:0;color:#dbe6ff}.credit-card{min-width:210px;padding:20px;border:1px solid #ffffff38;border-radius:16px;background:#ffffff14;display:grid}.credit-card strong{font-size:36px}.current-plan,.order-row{display:flex;justify-content:space-between;align-items:center;gap:20px;padding:18px 22px;border:1px solid var(--el-border-color-lighter);border-radius:14px;background:var(--el-bg-color)}.current-plan div,.order-row>div{display:flex;gap:10px;align-items:center}.current-plan div{flex-direction:column;align-items:flex-start}.product-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:16px}.product-card{padding:24px;border:1px solid var(--el-border-color-lighter);border-radius:16px;background:var(--el-bg-color)}.product-card p{min-height:44px;color:var(--el-text-color-secondary)}.price-row{display:flex;justify-content:space-between;align-items:end;padding-top:16px;border-top:1px solid var(--el-border-color-lighter)}.price-row>div{display:grid}.price-row strong{font-size:24px}.price-row small,.order-row span{color:var(--el-text-color-secondary)}.orders{display:grid;gap:12px}.order-row>div:last-child{flex-wrap:wrap;justify-content:flex-end}@media(max-width:700px){.billing-hero,.current-plan,.order-row{align-items:stretch;flex-direction:column}.credit-card{min-width:0}.order-row>div:last-child{justify-content:flex-start}}
</style>
