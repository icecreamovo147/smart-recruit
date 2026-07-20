<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatUnixDateTime } from '@shared/utils/format'
import {
  isAlipayReturnSearch,
  getPaymentReturnToken,
  detectAlipayScene,
  isBillingAlreadyPaidMessage,
  rememberBillingPendingOrder,
  sleep,
  stripAlipayReturnQuery,
  submitAlipayPayment,
  takeBillingPendingOrder,
} from '@shared/utils/billingReturn'
import {
  createBillingOrder,
  getBillingAccount,
  getBillingCatalog,
  listBillingOrders,
  payBillingOrder,
  refundBillingOrder,
  syncBillingPaymentReturn,
  type BillingAccount,
  type BillingOrder,
  type BillingProduct,
} from '@/api/billing'

const loading = ref(false)
const syncingPayment = ref(false)
const loadError = ref('')
const account = ref<BillingAccount | null>(null)
const products = ref<BillingProduct[]>([])
const orders = ref<BillingOrder[]>([])
const environment = ref('sandbox')
const activeSection = ref<'plans' | 'packs' | 'orders'>('plans')
const purchasingPriceId = ref(0)
const payingOrderNo = ref('')
const catalogRef = ref<HTMLElement | null>(null)
let syncAborted = false

const subscriptionProducts = computed(() => products.value.filter((item) => item.product_type === 'subscription'))
const creditPacks = computed(() => products.value.filter((item) => item.product_type === 'credit_pack'))
const scheduledSubscription = computed(() => account.value?.scheduled_subscription)
const usagePercent = computed(() => {
  const total = account.value?.total_credits || 0
  return total ? Math.min(100, Math.round(((account.value?.used_credits || 0) / total) * 100)) : 0
})

const subscriptionPurchaseLabel = (productKey: string) => {
  const scheduled = scheduledSubscription.value
  if (!scheduled) return '支付宝沙箱支付'
  return scheduled.product_key === productKey ? '已完成续费' : '已有待生效套餐'
}

const load = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const [catalog, current, history] = await Promise.all([
      getBillingCatalog(),
      getBillingAccount(),
      listBillingOrders(),
    ])
    products.value = catalog.products || []
    environment.value = catalog.payment_environment || 'sandbox'
    account.value = current
    orders.value = history.orders || []
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '套餐与额度加载失败'
  } finally {
    loading.value = false
  }
}

const errorMessage = (error: unknown) => (error instanceof Error ? error.message : String(error || ''))

const clearAlipayReturnUrl = () => {
  const next = stripAlipayReturnQuery(window.location.href)
  window.history.replaceState({}, document.title, next)
}

const syncOrderPayment = async (orderNo: string): Promise<'paid' | 'pending' | 'failed'> => {
  try {
    await payBillingOrder(orderNo, 'sync', { silentError: true })
    return 'paid'
  } catch (error) {
    const message = errorMessage(error)
    if (isBillingAlreadyPaidMessage(message)) return 'paid'
    if (message.includes('支付结果确认中')) return 'pending'
    return 'failed'
  }
}

const finalizeAlipayReturn = async (orderNo: string) => {
  if (!orderNo || syncAborted) return
  syncingPayment.value = true
  activeSection.value = 'orders'
  clearAlipayReturnUrl()
  try {
    for (let attempt = 0; attempt < 20 && !syncAborted; attempt += 1) {
      const syncResult = await syncOrderPayment(orderNo)
      await load()
      const order = orders.value.find((item) => item.order_no === orderNo)
      if (order?.status === 'paid' || syncResult === 'paid') {
        ElMessage.success('支付成功，套餐与订单状态已更新')
        return
      }
      if (syncResult === 'failed' && order && order.status !== 'paying' && order.status !== 'pending') {
        break
      }
      await sleep(1500)
    }
    ElMessage.warning('支付结果仍在确认中，请稍后点击刷新')
  } finally {
    syncingPayment.value = false
  }
}

const finalizePaymentReturnToken = async (token: string) => {
  if (!token || syncAborted) return
  syncingPayment.value = true
  activeSection.value = 'orders'
  clearAlipayReturnUrl()
  try {
    for (let attempt = 0; attempt < 20 && !syncAborted; attempt += 1) {
      try {
        await syncBillingPaymentReturn(token)
        await load()
        ElMessage.success('支付成功，套餐与订单状态已更新')
        return
      } catch (error) {
        if (!errorMessage(error).includes('支付结果确认中')) break
      }
      await sleep(1500)
    }
    await load()
    ElMessage.warning('支付结果仍在确认中，请稍后点击刷新')
  } finally { syncingPayment.value = false }
}

const purchase = async (product: BillingProduct, priceId: number, amountFen: number) => {
  if (amountFen <= 0) return
  if (product.product_type === 'subscription' && scheduledSubscription.value) {
    ElMessage.info('当前已有待生效套餐，无需重复购买')
    return
  }
  await ElMessageBox.confirm(
    `确认购买「${product.name}」，沙箱标价 ¥${(amountFen / 100).toFixed(2)}；升级订单将由服务端按剩余周期计算差价。`,
    '支付宝沙箱支付',
    { type: 'warning' },
  )
  purchasingPriceId.value = priceId
  try {
    const current = account.value?.subscription
    const hasPaidSubscription = current?.source === 'paid_subscription'
    const orderType = product.product_type === 'credit_pack'
      ? 'credit_pack'
      : (!hasPaidSubscription ? 'subscribe' : current.product_key === product.product_key ? 'renew' : 'upgrade')
    const created = await createBillingOrder(priceId, orderType)
    await continuePayment(created.order)
  } finally {
    purchasingPriceId.value = 0
  }
}

const continuePayment = async (order: BillingOrder) => {
  payingOrderNo.value = order.order_no
  try {
    const scene = detectAlipayScene()
    const payment = await payBillingOrder(order.order_no, scene)
    if (payment.payment_environment !== 'sandbox') throw new Error('当前开发版本只允许支付宝沙箱支付')
    rememberBillingPendingOrder(order.order_no)
    submitAlipayPayment(payment.redirect_url)
  } catch (error) {
    await load()
    if (isBillingAlreadyPaidMessage(errorMessage(error))) {
      ElMessage.success('订单已支付成功，套餐与订单状态已更新')
      return
    }
    throw error
  } finally {
    payingOrderNo.value = ''
  }
}

const refund = async (order: BillingOrder) => {
  const { value } = await ElMessageBox.prompt(
    '请填写退款原因；已消费额度的订单会转人工审核。',
    '申请退款',
    { inputPattern: /\S+/, inputErrorMessage: '退款原因不能为空' },
  )
  await refundBillingOrder(order.order_no, value)
  ElMessage.success('退款申请已提交')
  await load()
}

const formatTime = (value?: unknown) => formatUnixDateTime(value)
const formatTerm = (term: string) => term === 'yearly' ? '年' : term === 'monthly' ? '月' : '次'
const orderStatusLabel = (status: string) => ({
  pending: '待支付', paying: '支付中', paid: '已支付', closed: '已关闭',
  refunding: '退款中', refunded: '已退款',
}[status] || status)
const orderStatusType = (status: string) => {
  if (status === 'paid') return 'success'
  if (status === 'pending' || status === 'paying' || status === 'refunding') return 'warning'
  return 'info'
}

const showSection = async (section: 'plans' | 'packs' | 'orders') => {
  activeSection.value = section
  await nextTick()
  catalogRef.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

onMounted(async () => {
  const returnedFromAlipay = isAlipayReturnSearch(window.location.search)
  const returnToken = getPaymentReturnToken(window.location.search)
  const pendingOrderNo = takeBillingPendingOrder()
  await load()
  if (returnToken) {
    await finalizePaymentReturnToken(returnToken)
  } else if (returnedFromAlipay || pendingOrderNo) {
    const orderNo = pendingOrderNo
      || orders.value.find((item) => item.status === 'paying' || item.status === 'pending')?.order_no
      || ''
    if (orderNo) await finalizeAlipayReturn(orderNo)
    else clearAlipayReturnUrl()
  }
})
onBeforeUnmount(() => {
  syncAborted = true
})
</script>

<template>
  <section class="billing-page" v-loading="loading || syncingPayment">
    <el-alert
      v-if="environment === 'sandbox'"
      title="当前为支付宝沙箱环境：所有金额和交易均为测试数据，不会产生真实扣款。"
      type="warning"
      :closable="false"
      show-icon
    />
    <el-alert
      v-if="scheduledSubscription"
      :title="`续费已完成：${scheduledSubscription.product_name} 将于 ${formatTime(scheduledSubscription.current_period_start_unix_ms)} 自动生效`"
      :description="`有效期至 ${formatTime(scheduledSubscription.current_period_end_unix_ms)}，无需重复购买。`"
      type="success"
      :closable="false"
      show-icon
    />
    <el-alert v-if="loadError" :title="loadError" type="error" :closable="false" show-icon>
      <template #default><el-button link type="primary" @click="load">重新加载</el-button></template>
    </el-alert>

    <header class="billing-hero">
      <div class="hero-copy">
        <span>AI BILLING</span>
        <h1>AI 套餐与额度</h1>
        <p>按月获得 AI 求职额度，需要更多时可购买长期有效的加量包。</p>
        <div class="current-plan-line">
          <el-tag effect="dark" round>{{ account?.subscription?.source === 'free_tier' ? '免费权益' : '当前套餐' }}</el-tag>
          <strong>{{ account?.subscription?.product_name || '尚未开通' }}</strong>
          <span v-if="account?.next_refresh_at_unix_ms">{{ formatTime(account.next_refresh_at_unix_ms) }} 刷新</span>
        </div>
        <div class="hero-actions"><el-button class="hero-pay-button" size="small" @click="showSection('plans')">购买套餐</el-button><el-button class="hero-pay-button" size="small" plain @click="showSection('packs')">购买加量包</el-button></div>
      </div>
      <div class="credit-card">
        <small>可用 AI 额度</small>
        <strong>{{ (account?.available_credits || 0).toLocaleString() }}</strong>
        <el-progress :percentage="usagePercent" :stroke-width="7" :show-text="false" color="#fff" />
        <div><span>已使用 {{ (account?.used_credits || 0).toLocaleString() }}</span><span>已预占 {{ (account?.reserved_credits || 0).toLocaleString() }}</span></div>
      </div>
    </header>

    <section ref="catalogRef" class="billing-workspace">
      <header class="workspace-heading">
        <div><h2>套餐与订单</h2><p>在一个工作区内切换套餐、加量包和订单记录。</p></div>
        <el-button :loading="loading || syncingPayment" @click="load">刷新</el-button>
      </header>

      <el-tabs v-model="activeSection" class="billing-tabs">
        <el-tab-pane label="月度套餐" name="plans">
          <div v-if="subscriptionProducts.length" class="product-grid">
            <article v-for="product in subscriptionProducts" :key="product.id" class="product-card" :class="{ 'product-card--current': account?.subscription?.product_key === product.product_key }">
              <div class="product-head">
                <div class="product-title"><h3>{{ product.name }}</h3><el-tag v-if="account?.subscription?.product_key === product.product_key" type="success" effect="plain" size="small">当前权益</el-tag></div>
                <span>{{ product.description }}</span>
              </div>
              <div v-for="price in product.prices" :key="price.id" class="price-row">
                <div><strong>{{ price.amount_fen ? `¥${(price.amount_fen / 100).toFixed(2)}` : '免费' }}</strong><span>/ {{ formatTerm(price.billing_term) }}</span><small>每月含 {{ price.included_credits.toLocaleString() }} AI 额度</small></div>
                <el-button v-if="price.amount_fen > 0" type="primary" :disabled="Boolean(scheduledSubscription)" :loading="purchasingPriceId === price.id" @click="purchase(product, price.id, price.amount_fen)">{{ subscriptionPurchaseLabel(product.product_key) }}</el-button>
                <el-button v-else disabled>{{ account?.subscription?.product_key === product.product_key ? '当前使用中' : '免费开放' }}</el-button>
              </div>
            </article>
          </div>
          <el-empty v-else description="平台暂未发布可选套餐" />
        </el-tab-pane>

        <el-tab-pane label="AI 加量包" name="packs">
          <div v-if="creditPacks.length" class="product-grid">
            <article v-for="product in creditPacks" :key="product.id" class="product-card product-card--pack">
              <div class="product-head"><div class="product-title"><h3>{{ product.name }}</h3><el-tag type="success" effect="plain" size="small">一次性加量</el-tag></div><span>{{ product.description }}</span></div>
              <div v-for="price in product.prices" :key="price.id" class="price-row"><div><strong>¥{{ (price.amount_fen / 100).toFixed(2) }}</strong><small>{{ price.included_credits.toLocaleString() }} AI 额度 · 十二个月有效</small></div><el-button type="primary" plain :loading="purchasingPriceId === price.id" @click="purchase(product, price.id, price.amount_fen)">支付宝沙箱支付</el-button></div>
            </article>
          </div>
          <el-empty v-else description="暂时没有可购买的加量包" />
        </el-tab-pane>

        <el-tab-pane :label="`订单记录${orders.length ? ` (${orders.length})` : ''}`" name="orders">
          <el-table :data="orders" class="order-table" empty-text="暂无订单">
            <el-table-column label="商品" min-width="220"><template #default="{ row }"><div class="order-product"><strong>{{ row.product_name }}</strong><span>{{ row.order_no }}</span></div></template></el-table-column>
            <el-table-column label="金额" width="130" align="right"><template #default="{ row }">¥{{ (row.amount_fen / 100).toFixed(2) }}</template></el-table-column>
            <el-table-column label="创建时间" width="180"><template #default="{ row }">{{ formatTime(row.created_at_unix_ms) }}</template></el-table-column>
            <el-table-column label="状态" width="110" align="center"><template #default="{ row }"><el-tag :type="orderStatusType(row.status)" size="small">{{ orderStatusLabel(row.status) }}</el-tag></template></el-table-column>
            <el-table-column label="操作" width="130" align="center"><template #default="{ row }"><el-button v-if="row.status === 'pending' || row.status === 'paying'" link type="primary" :loading="payingOrderNo === row.order_no" @click="continuePayment(row)">继续支付</el-button><el-button v-else-if="row.status === 'paid'" link type="danger" @click="refund(row)">申请退款</el-button><span v-else>—</span></template></el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </section>
  </section>
</template>

<style scoped>
.billing-page{min-width:0;display:grid;gap:14px;padding:0 0 24px}.billing-hero{display:grid;grid-template-columns:minmax(0,1fr) 210px;gap:20px;padding:19px 26px;border-radius:16px;background:linear-gradient(135deg,#14213d,#2457d6);color:#fff;box-shadow:0 10px 28px rgba(23,55,122,.14)}.hero-copy>span{font-size:11px;letter-spacing:.1em;opacity:.76}.billing-hero h1{margin:3px 0;font-size:26px;line-height:1.25}.billing-hero p{margin:0;color:#dbe6ff;font-size:13px}.current-plan-line{display:flex;align-items:center;gap:8px;margin-top:11px}.current-plan-line span{color:#dbe6ff;font-size:11px}.hero-actions{display:flex;gap:8px;margin-top:10px}.hero-pay-button{border-color:#ffffff55;background:#ffffff18;color:#fff}.hero-pay-button:hover{border-color:#fff;background:#fff;color:#2457d6}.credit-card{align-self:stretch;padding:13px 15px;border:1px solid #ffffff38;border-radius:12px;background:#ffffff14;display:grid;gap:5px}.credit-card>small{font-size:11px}.credit-card>strong{font-size:30px;line-height:1}.credit-card>div{display:flex;justify-content:space-between;gap:10px;color:#e7eeff;font-size:10px}.billing-workspace{min-width:0;padding:22px 26px 24px;border:1px solid var(--el-border-color-lighter);border-radius:16px;background:var(--el-bg-color)}.workspace-heading{display:flex;justify-content:space-between;align-items:flex-start;gap:20px}.workspace-heading h2{margin:0 0 4px;font-size:21px}.workspace-heading p,.product-head>span{margin:0;color:var(--el-text-color-secondary);font-size:13px}.billing-tabs{min-width:0;margin-top:8px}.billing-tabs :deep(.el-tabs__header){margin-bottom:18px}.billing-tabs :deep(.el-tabs__content),.billing-tabs :deep(.el-tab-pane){min-width:0;max-width:100%}.product-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}.product-card{display:flex;min-height:190px;min-width:0;flex-direction:column;justify-content:space-between;padding:18px;border:1px solid var(--el-border-color-lighter);border-radius:12px;background:var(--el-bg-color)}.product-card--current{border-color:#80aaff;box-shadow:0 8px 20px rgba(64,112,214,.08)}.product-card--pack{min-height:175px}.product-title{display:flex;align-items:center;justify-content:space-between;gap:10px}.product-head h3{margin:0 0 7px;font-size:19px}.price-row{display:flex;justify-content:space-between;align-items:end;gap:14px;padding-top:15px;border-top:1px solid var(--el-border-color-lighter)}.price-row>div{display:grid;grid-template-columns:auto auto;align-items:end;gap:2px 5px}.price-row strong{font-size:23px}.price-row span,.price-row small,.order-product span,.order-action-hint{color:var(--el-text-color-secondary)}.price-row small{grid-column:1/-1;font-size:12px}.order-action-hint{font-size:12px}.order-table{width:100%;max-width:100%;border:1px solid var(--el-border-color-lighter);border-radius:9px}.order-product{display:grid;gap:3px}.order-product span{font-size:12px}@media(max-width:900px){.product-grid{grid-template-columns:1fr}}@media(max-width:800px){.billing-page{padding-bottom:16px}.billing-hero{grid-template-columns:1fr;padding:20px}.current-plan-line{align-items:flex-start;flex-direction:column}.hero-actions{flex-wrap:wrap}.billing-workspace{padding:20px}.workspace-heading{align-items:stretch;flex-direction:column}.price-row{align-items:stretch;flex-direction:column}.credit-card{min-width:0}}
</style>
