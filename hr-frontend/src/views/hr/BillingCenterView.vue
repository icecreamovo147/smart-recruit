<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { formatUnixDateTime, parseUnixTimestamp } from '@shared/utils/format'
import {
  isAlipayReturnSearch,
  isBillingAlreadyPaidMessage,
  rememberBillingPendingOrder,
  sleep,
  stripAlipayReturnQuery,
  takeBillingPendingOrder,
} from '@shared/utils/billingReturn'
import { billingOrderRemainingLabel, effectiveBillingOrderStatus, isBillingOrderPayable } from '@/utils/billingOrder'
import {
  createBillingOrder,
  getBillingAccount,
  getBillingCatalog,
  listBillingOrders,
  payBillingOrder,
  refundBillingOrder,
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
const currentTimeMs = ref(Date.now())
let clockTimer: number | undefined
let syncAborted = false

const subscriptionProducts = computed(() => products.value.filter((item) => item.product_type === 'subscription'))
const creditPacks = computed(() => products.value.filter((item) => item.product_type === 'credit_pack'))
const usagePercent = computed(() => {
  const total = account.value?.total_credits || 0
  return total ? Math.min(100, Math.round(((account.value?.used_credits || 0) / total) * 100)) : 0
})
const planSourceLabel = computed(() => {
  const source = account.value?.subscription?.source
  if (source === 'platform_plan') return '平台分配'
  if (source === 'free_tier') return '免费权益'
  if (source === 'paid_subscription') return '付费订阅'
  return '未配置'
})

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

const isOrderPayable = (order: BillingOrder) => (
  isBillingOrderPayable(order, currentTimeMs.value)
)

const effectiveOrderStatus = (order: BillingOrder) => effectiveBillingOrderStatus(order, currentTimeMs.value)
const formatRemaining = (order: BillingOrder) => billingOrderRemainingLabel(order, currentTimeMs.value)

const latestPendingOrder = (source: BillingOrder[]) => source.find(isOrderPayable)

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
      if (syncResult === 'failed' && order && !isOrderPayable(order) && order.status !== 'paying' && order.status !== 'pending') {
        break
      }
      await sleep(1500)
    }
    ElMessage.warning('支付结果仍在确认中，请稍后点击刷新')
  } finally {
    syncingPayment.value = false
  }
}

const purchase = async (product: BillingProduct, priceId: number, amountFen: number) => {
  purchasingPriceId.value = priceId
  try {
    const history = await listBillingOrders()
    orders.value = history.orders || []
    const pendingOrder = latestPendingOrder(orders.value)
    let replacePendingOrder = false
    if (pendingOrder) {
      try {
        await ElMessageBox.confirm(
          `当前有待支付订单「${pendingOrder.product_name}」（¥${(pendingOrder.amount_fen / 100).toFixed(2)}），剩余 ${formatRemaining(pendingOrder)}。你可以继续支付该订单，或关闭它并为「${product.name}」创建新订单。`,
          '发现待支付订单',
          {
            type: 'warning',
            confirmButtonText: '继续支付原订单',
            cancelButtonText: '创建新订单',
            distinguishCancelAndClose: true,
            closeOnClickModal: false,
            closeOnPressEscape: false,
          },
        )
        await continuePayment(pendingOrder)
        return
      } catch (action) {
        if (action !== 'cancel') return
        replacePendingOrder = true
      }
    } else {
      await ElMessageBox.confirm(
        `确认购买「${product.name}」，沙箱标价 ¥${(amountFen / 100).toFixed(2)}；订单创建后需在 30 分钟内完成支付。`,
        '支付宝沙箱支付',
        { type: 'warning', confirmButtonText: '创建订单并支付' },
      )
    }
    const current = account.value?.subscription
    const hasPaidSubscription = current?.source === 'paid_subscription'
    const orderType = product.product_type === 'credit_pack'
      ? 'credit_pack'
      : (!hasPaidSubscription ? 'subscribe' : current.product_key === product.product_key ? 'renew' : 'upgrade')
    const created = await createBillingOrder(priceId, orderType, replacePendingOrder)
    await continuePayment(created.order)
  } finally {
    purchasingPriceId.value = 0
  }
}

const continuePayment = async (order: BillingOrder) => {
  currentTimeMs.value = Date.now()
  if (!isOrderPayable(order)) {
    ElMessage.warning('订单已超过支付时限并关闭，请重新创建订单')
    await load()
    return
  }
  payingOrderNo.value = order.order_no
  try {
    const scene = window.matchMedia('(max-width: 768px)').matches ? 'wap' : 'desktop'
    const payment = await payBillingOrder(order.order_no, scene)
    if (payment.payment_environment !== 'sandbox') throw new Error('当前开发版本只允许支付宝沙箱支付')
    rememberBillingPendingOrder(order.order_no)
    window.location.assign(payment.redirect_url)
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
const formatExpiry = (value?: unknown) => parseUnixTimestamp(value) === null ? '暂无到期额度' : formatUnixDateTime(value)
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
  clockTimer = window.setInterval(() => { currentTimeMs.value = Date.now() }, 1000)
  const returnedFromAlipay = isAlipayReturnSearch(window.location.search)
  const pendingOrderNo = takeBillingPendingOrder()
  await load()
  if (returnedFromAlipay || pendingOrderNo) {
    const orderNo = pendingOrderNo
      || orders.value.find((item) => item.status === 'paying' || item.status === 'pending')?.order_no
      || ''
    if (orderNo) await finalizeAlipayReturn(orderNo)
    else clearAlipayReturnUrl()
  }
})
onBeforeUnmount(() => {
  syncAborted = true
  if (clockTimer !== undefined) window.clearInterval(clockTimer)
})
</script>

<template>
  <section class="console-page console-page--fill billing-page">
    <div class="workspace-surface billing-workspace" v-loading="loading || syncingPayment">
      <header class="workspace-surface__header">
        <div class="workspace-surface__header-copy">
          <p class="console-eyebrow">AI BILLING &amp; ENTITLEMENTS</p>
          <h1 class="console-title">AI 套餐与额度</h1>
          <p class="console-description">查看企业当前 AI 权益和额度消耗，管理可购买套餐、加量包及订单记录。</p>
        </div>
        <div class="workspace-surface__header-actions">
          <el-button :icon="Refresh" :loading="loading || syncingPayment" @click="load">刷新</el-button>
        </div>
      </header>

      <div class="workspace-surface__divider"></div>

      <div class="workspace-surface__body billing-workspace__body">
        <div class="billing-content">
          <el-alert
            v-if="syncingPayment"
            title="正在确认支付宝支付结果并刷新套餐与订单…"
            type="info"
            :closable="false"
            show-icon
          />
          <el-alert
            v-if="environment === 'sandbox'"
            title="当前为支付宝沙箱环境，交易仅用于联调，不会产生真实扣款。"
            type="warning"
            :closable="false"
            show-icon
          />
          <el-alert v-if="loadError" :title="loadError" type="error" :closable="false" show-icon>
            <template #default><el-button link type="primary" @click="load">重新加载</el-button></template>
          </el-alert>

          <section class="console-card entitlement-card">
            <div class="console-card__head">
              <div>
                <h2 class="console-card__title">当前企业权益</h2>
                <p class="console-card__desc">套餐来源、额度刷新周期及本期使用情况。</p>
              </div>
              <div class="entitlement-actions"><el-tag effect="plain">{{ planSourceLabel }}</el-tag><el-button type="primary" size="small" @click="showSection('plans')">购买套餐</el-button></div>
            </div>
            <div class="entitlement-card__body">
              <div class="current-plan">
                <span class="current-plan__label">当前套餐</span>
                <strong>{{ account?.subscription?.product_name || '尚未配置套餐' }}</strong>
                <p v-if="account?.subscription">
                  每月包含 {{ (account.subscription.included_credits || 0).toLocaleString() }} AI 额度
                </p>
                <p v-else>请联系平台管理员分配企业套餐，或从套餐列表选择可购买方案。</p>
              </div>
              <div class="usage-overview">
                <div class="usage-overview__head">
                  <span>本期额度使用</span>
                  <strong>{{ usagePercent }}%</strong>
                </div>
                <el-progress :percentage="usagePercent" :stroke-width="8" :show-text="false" />
                <div class="usage-overview__meta">
                  <span>下次刷新：{{ formatTime(account?.next_refresh_at_unix_ms) }}</span>
                  <span>最近到期：{{ formatExpiry(account?.next_expiry_at_unix_ms) }}</span>
                </div>
              </div>
            </div>
          </section>

          <div class="console-stats billing-stats">
            <article class="console-stat"><div class="console-stat__label">可用额度</div><div class="console-stat__value">{{ (account?.available_credits || 0).toLocaleString() }}</div><div class="console-stat__hint">当前可立即使用</div></article>
            <article class="console-stat"><div class="console-stat__label">本期总额度</div><div class="console-stat__value">{{ (account?.total_credits || 0).toLocaleString() }}</div><div class="console-stat__hint">套餐与加量包合计</div></article>
            <article class="console-stat"><div class="console-stat__label">已使用</div><div class="console-stat__value">{{ (account?.used_credits || 0).toLocaleString() }}</div><div class="console-stat__hint">已完成结算的消耗</div></article>
            <article class="console-stat"><div class="console-stat__label">已预占</div><div class="console-stat__value">{{ (account?.reserved_credits || 0).toLocaleString() }}</div><div class="console-stat__hint">执行中的 AI 任务</div></article>
          </div>

          <section ref="catalogRef" class="console-card billing-catalog">
            <div class="console-card__head billing-catalog__head">
              <div>
                <h2 class="console-card__title">套餐与订单</h2>
                <p class="console-card__desc">选择套餐或加量包，并查看当前企业的沙箱订单。</p>
              </div>
              <el-tag v-if="environment === 'sandbox'" type="warning" effect="plain">支付宝沙箱</el-tag>
            </div>

            <el-tabs v-model="activeSection" class="billing-tabs">
              <el-tab-pane label="企业套餐" name="plans">
                <div v-if="subscriptionProducts.length" class="product-grid">
                  <article v-for="product in subscriptionProducts" :key="product.id" class="product-card">
                    <div class="product-card__head"><div><span class="product-card__type">周期订阅</span><h3>{{ product.name }}</h3></div><p>{{ product.description }}</p></div>
                    <div v-for="price in product.prices" :key="price.id" class="price-row">
                      <div><strong>{{ price.amount_fen ? `¥${(price.amount_fen / 100).toFixed(2)}` : '免费' }}</strong><span>/ {{ formatTerm(price.billing_term) }}</span><small>{{ price.included_credits.toLocaleString() }} AI 额度</small></div>
                      <el-button v-if="price.amount_fen > 0" type="primary" :loading="purchasingPriceId === price.id" @click="purchase(product, price.id, price.amount_fen)">支付宝沙箱支付</el-button>
                    </div>
                  </article>
                </div>
                <el-empty v-else class="console-empty" description="平台暂未发布可购买的企业套餐" />
              </el-tab-pane>

              <el-tab-pane label="AI 加量包" name="packs">
                <div v-if="creditPacks.length" class="product-grid">
                  <article v-for="product in creditPacks" :key="product.id" class="product-card">
                    <div class="product-card__head"><div><span class="product-card__type product-card__type--pack">一次性加量</span><h3>{{ product.name }}</h3></div><p>{{ product.description }}</p></div>
                    <div v-for="price in product.prices" :key="price.id" class="price-row">
                      <div><strong>¥{{ (price.amount_fen / 100).toFixed(2) }}</strong><small>{{ price.included_credits.toLocaleString() }} AI 额度 · 十二个月有效</small></div>
                      <el-button type="primary" plain :loading="purchasingPriceId === price.id" @click="purchase(product, price.id, price.amount_fen)">支付宝沙箱支付</el-button>
                    </div>
                  </article>
                </div>
                <el-empty v-else class="console-empty" description="平台暂未发布企业 AI 加量包" />
              </el-tab-pane>

              <el-tab-pane :label="`订单记录 ${orders.length ? `(${orders.length})` : ''}`" name="orders">
                <el-table :data="orders" class="console-table order-table" empty-text="暂无订单">
                  <el-table-column label="商品" min-width="220">
                    <template #default="{ row }"><div class="console-entity"><div class="console-entity__name">{{ row.product_name }}</div><div class="console-entity__meta">{{ row.order_no }}</div></div></template>
                  </el-table-column>
                  <el-table-column label="金额" width="130" align="right"><template #default="{ row }">¥{{ (row.amount_fen / 100).toFixed(2) }}</template></el-table-column>
                  <el-table-column label="创建时间" width="180"><template #default="{ row }">{{ formatTime(row.created_at_unix_ms) }}</template></el-table-column>
                  <el-table-column label="支付截止" width="200"><template #default="{ row }"><div class="order-expiry"><span>{{ formatTime(row.expires_at_unix_ms) }}</span><small v-if="isOrderPayable(row)">剩余 {{ formatRemaining(row) }}</small><small v-else-if="effectiveOrderStatus(row) === 'closed'">已关闭，无法继续支付</small></div></template></el-table-column>
                  <el-table-column label="状态" width="110" align="center"><template #default="{ row }"><el-tag :type="orderStatusType(effectiveOrderStatus(row))" size="small">{{ orderStatusLabel(effectiveOrderStatus(row)) }}</el-tag></template></el-table-column>
                  <el-table-column label="操作" width="130" align="center"><template #default="{ row }"><el-button v-if="isOrderPayable(row)" link type="primary" :loading="payingOrderNo === row.order_no" @click="continuePayment(row)">继续支付</el-button><el-button v-else-if="row.status === 'paid'" link type="danger" @click="refund(row)">申请退款</el-button><span v-else class="table-placeholder">—</span></template></el-table-column>
                </el-table>
              </el-tab-pane>
            </el-tabs>
          </section>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.billing-page{padding-bottom:0;overflow:hidden}.billing-workspace{height:100%;min-width:0}.billing-workspace__body{min-width:0;padding:0;overflow:auto}.billing-content{min-width:0;display:grid;gap:16px;padding:18px 24px 16px}.billing-content>*{min-width:0}.entitlement-actions{display:flex;align-items:center;gap:8px}.entitlement-card__body{display:grid;grid-template-columns:minmax(260px,.75fr) minmax(360px,1.25fr);gap:32px;align-items:center;padding:22px 24px}.current-plan{display:grid;gap:6px}.current-plan__label{color:var(--text-muted);font-size:12px}.current-plan>strong{color:var(--text-primary);font-size:26px;line-height:1.25}.current-plan p{margin:0;color:var(--text-muted);font-size:13px;line-height:1.6}.usage-overview{display:grid;gap:10px}.usage-overview__head,.usage-overview__meta{display:flex;justify-content:space-between;gap:16px}.usage-overview__head{align-items:end;color:var(--text-muted);font-size:13px}.usage-overview__head strong{color:var(--text-primary);font-size:18px}.usage-overview__meta{color:var(--text-faint);font-size:12px}.billing-stats{margin-bottom:0}.billing-catalog{min-width:0;overflow:hidden}.billing-catalog__head{border-bottom:0}.billing-tabs{min-width:0;padding:0 18px 18px}.billing-tabs :deep(.el-tabs__header){margin-bottom:18px}.billing-tabs :deep(.el-tabs__content),.billing-tabs :deep(.el-tab-pane){min-width:0;max-width:100%}.product-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(300px,1fr));gap:14px}.product-card{display:flex;min-height:205px;min-width:0;flex-direction:column;justify-content:space-between;padding:18px;border:1px solid var(--border);border-radius:10px;background:var(--surface)}.product-card__head h3{margin:8px 0 6px;color:var(--text-primary);font-size:18px}.product-card__head p{margin:0;color:var(--text-muted);font-size:13px;line-height:1.55}.product-card__type{display:inline-flex;padding:3px 8px;border-radius:6px;color:var(--el-color-primary);background:var(--el-color-primary-light-9);font-size:11px;font-weight:600}.product-card__type--pack{color:var(--el-color-success);background:var(--el-color-success-light-9)}.price-row{display:flex;justify-content:space-between;align-items:end;gap:14px;padding-top:16px;border-top:1px solid var(--border)}.price-row>div{display:grid;grid-template-columns:auto auto;align-items:end;gap:2px 5px}.price-row strong{color:var(--text-primary);font-size:23px}.price-row span,.price-row small{color:var(--text-muted)}.price-row small{grid-column:1/-1;font-size:12px}.order-table{width:100%;max-width:100%;border:1px solid var(--border);border-radius:8px}.order-expiry{display:grid;gap:3px}.order-expiry small{color:var(--text-faint);font-size:11px}.table-placeholder{color:var(--text-faint)}@media(max-width:900px){.entitlement-card__body{grid-template-columns:1fr}.billing-stats{grid-template-columns:repeat(2,minmax(0,1fr))}}@media(max-width:720px){.billing-content{padding:14px 14px 10px}.entitlement-card__body{padding:18px}.usage-overview__meta{flex-direction:column;gap:4px}.billing-stats{grid-template-columns:1fr}.price-row{align-items:stretch;flex-direction:column}.price-row>div{justify-content:start}}
</style>
