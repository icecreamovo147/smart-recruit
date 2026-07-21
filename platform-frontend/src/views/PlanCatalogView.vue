<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listAIRateCards, listBillingProducts, listBillingRefunds, listPlans, publishPlanVersion, reviewBillingRefund, saveAIRateCard, saveBillingPrice, savePlanVersion, type AIRateCardAdmin, type BillingProductAdmin, type BillingRefundAdmin } from '@/api/control'
import { listModels, listProviders } from '@/api/llm'
import { listPlatformAICapabilities, type PlatformAICapability } from '@/api/platformAI'
import { PLATFORM_PERMISSIONS } from '@/permissions'
import { useAuthStore } from '@/stores/auth'
import type { PlatformEntitlement, PlatformPlan, PlatformPlanVersion } from '@/types'
import type { LlmModel, LlmProvider } from '@shared/types/llm'
import { formatShanghaiDateTime, toShanghaiRFC3339 } from '@shared/utils/format'
import { enabledRateModelsForProvider, enabledRateProviders, isEnabledRateTarget } from './rateCardCatalog'

const auth = useAuthStore()
const loading = ref(false)
type CatalogSection = 'plans' | 'products' | 'rates' | 'refunds'
const activeSection = ref<CatalogSection>('plans')
const plans = ref<PlatformPlan[]>([])
const billingProducts = ref<BillingProductAdmin[]>([])
const rateCards = ref<AIRateCardAdmin[]>([])
const rateProviders = ref<LlmProvider[]>([])
const rateModels = ref<LlmModel[]>([])
const rateTargetsLoaded = ref(false)
const refunds = ref<BillingRefundAdmin[]>([])
const aiCapabilities = ref<PlatformAICapability[]>([])
const paymentEnvironment = ref('sandbox')
const canManage = computed(() => auth.can(PLATFORM_PERMISSIONS.PLAN_MANAGE))
const canPublish = computed(() => auth.can(PLATFORM_PERMISSIONS.PLAN_PUBLISH))
const canReviewRefund = computed(() => auth.can(PLATFORM_PERMISSIONS.BILLING_REFUND_REVIEW))
const editorVisible = ref(false)
const publishVisible = ref(false)
const selectedPlan = ref<PlatformPlan | null>(null)
const selectedVersion = ref<PlatformPlanVersion | null>(null)
const form = reactive({
  version_id: 0, change_note: '', members: 1, jobs: 1, applications: 1, resumes: 1,
  aiHr: true, aiChat: true, aiResumeParse: true, aiMatchEvaluation: true,
  aiApplicationAnalysis: true, aiAgentRun: true, aiMonthlyCredits: 100,
  aiConcurrentRuns: 1, aiSingleRunCredits: 20,
})
const publishForm = reactive({ effective_at: '', reason: '' })
const priceVisible = ref(false)
const selectedBillingProduct = ref<BillingProductAdmin | null>(null)
const priceForm = reactive({ price_version_id: 0, billing_term: 'monthly', amount_yuan: 1, included_credits: 100 })
const rateVisible = ref(false)
const editingRateTarget = ref(false)
const rateForm = reactive({ provider_key: '', model_key: '', input_yuan: 0, output_yuan: 0, cached_yuan: 0, credit_yuan: 0.01 })
const rateProviderOptions = computed(() => enabledRateProviders(rateProviders.value))
const rateModelOptions = computed(() => enabledRateModelsForProvider(rateProviders.value, rateModels.value, rateForm.provider_key))

const loadRateTargets = async () => {
  if (rateTargetsLoaded.value) return
  const [providerResult, modelResult] = await Promise.all([listProviders(1, 500), listModels(1, 500)])
  rateProviders.value = providerResult.list || []
  rateModels.value = modelResult.list || []
  rateTargetsLoaded.value = true
}

const entitlementLabels: Record<string, string> = {
  'members.max': '有效成员上限', 'jobs.published.max': '在线岗位上限',
  'applications.monthly.max': '月投递上限', 'resumes.storage.max': '简历存储上限',
  'ai.hr.enabled': 'HR AI 总开关', 'ai.chat.enabled': 'AI 对话',
  'ai.resume_parse.enabled': '简历解析', 'ai.match_evaluation.enabled': '匹配评估',
  'ai.application_analysis.enabled': '申请分析', 'ai.agent_run.enabled': 'Agent 任务',
  'ai.chat.release_version_id': 'AI 对话能力版本',
  'ai.resume_parse.release_version_id': '简历解析能力版本',
  'ai.match_evaluation.release_version_id': '匹配评估能力版本',
  'ai.application_analysis.release_version_id': '申请分析能力版本',
  'ai.agent_run.release_version_id': 'Agent 能力版本',
  'ai.credits.monthly': '每月 AI 额度', 'ai.concurrent_runs.max': 'AI 并发任务',
  'ai.single_run.max_credits': '单次任务额度上限',
}

const load = async (section: CatalogSection = activeSection.value) => {
  loading.value = true
  try {
    if (section === 'plans') {
      const [planResult, capabilityResult] = await Promise.all([listPlans(), listPlatformAICapabilities()])
      plans.value = planResult.list || []
      aiCapabilities.value = (capabilityResult.list || []).filter((item) => item.audience === 'tenant_hr')
    } else if (section === 'products') {
      const billingResult = await listBillingProducts()
      billingProducts.value = billingResult.products || []
      paymentEnvironment.value = billingResult.payment_environment || 'sandbox'
    } else if (section === 'rates') {
      const [rateResult] = await Promise.all([listAIRateCards(), loadRateTargets()])
      rateCards.value = rateResult.rates || []
    } else {
      const refundResult = await listBillingRefunds()
      refunds.value = refundResult.refunds || []
    }
  } finally { loading.value = false }
}

const reviewRefund = async (refund: BillingRefundAdmin, action: 'approve' | 'reject') => {
  let reason = ''
  if (action === 'reject') {
    const result = await ElMessageBox.prompt('请填写拒绝退款的原因', '拒绝退款', { inputPattern: /\S+/, inputErrorMessage: '拒绝原因不能为空' })
    reason = result.value
  } else {
    await ElMessageBox.confirm(`确认批准订单 ${refund.order_no} 退款 ¥${(refund.amount_fen / 100).toFixed(2)}？`, '批准退款', { type: 'warning' })
  }
  await reviewBillingRefund(refund.refund_no, action, reason)
  ElMessage.success(action === 'approve' ? '退款已批准并提交支付宝' : '退款已拒绝')
  await load('refunds')
}

const switchSection = (value: string | number) => {
  activeSection.value = String(value) as CatalogSection
  void load(activeSection.value)
}

const openRateEditor = async (rate?: AIRateCardAdmin) => {
  await loadRateTargets()
  if (rate && !isEnabledRateTarget(rateProviders.value, rateModels.value, rate.provider_key, rate.model_key)) {
    ElMessage.error('该费率卡对应的供应商或模型已停用，请先在 LLM 配置中启用后再创建新版本')
    return
  }
  editingRateTarget.value = Boolean(rate)
  rateForm.provider_key = rate?.provider_key || ''
  rateForm.model_key = rate?.model_key || ''
  rateForm.input_yuan = (rate?.input_micros_per_1k_tokens || 0) / 1_000_000
  rateForm.output_yuan = (rate?.output_micros_per_1k_tokens || 0) / 1_000_000
  rateForm.cached_yuan = (rate?.cached_input_micros_per_1k_tokens || 0) / 1_000_000
  rateForm.credit_yuan = (rate?.credit_micros || 10_000) / 1_000_000
  rateVisible.value = true
}

const changeRateProvider = () => {
  if (!editingRateTarget.value) rateForm.model_key = ''
}

const submitRate = async () => {
  if (!isEnabledRateTarget(rateProviders.value, rateModels.value, rateForm.provider_key, rateForm.model_key)) {
    ElMessage.error('请选择当前已启用且相互匹配的供应商和模型')
    return
  }
  if (rateForm.credit_yuan <= 0 || (rateForm.input_yuan <= 0 && rateForm.output_yuan <= 0)) {
    ElMessage.error('请填写有效的供应商成本和额度换算价格')
    return
  }
  await saveAIRateCard({
    provider_key: rateForm.provider_key.trim(), model_key: rateForm.model_key.trim(),
    input_micros_per_1k_tokens: Math.round(rateForm.input_yuan * 1_000_000),
    output_micros_per_1k_tokens: Math.round(rateForm.output_yuan * 1_000_000),
    cached_input_micros_per_1k_tokens: Math.round(rateForm.cached_yuan * 1_000_000),
    credit_micros: Math.round(rateForm.credit_yuan * 1_000_000),
  })
  rateVisible.value = false
  ElMessage.success('费率卡已保存并发布')
  await load()
}

const openPriceEditor = (product: BillingProductAdmin) => {
  const latest = product.prices[0]
  selectedBillingProduct.value = product
  // Published versions are immutable; every UI edit creates a new version.
  priceForm.price_version_id = 0
  priceForm.billing_term = latest?.billing_term || (product.product_type === 'credit_pack' ? 'one_time' : 'monthly')
  priceForm.amount_yuan = latest ? latest.amount_fen / 100 : 1
  priceForm.included_credits = latest?.included_credits || 100
  priceVisible.value = true
}

const submitPrice = async () => {
  if (!selectedBillingProduct.value || priceForm.amount_yuan <= 0 || priceForm.included_credits <= 0) return
  await saveBillingPrice({ product_id: selectedBillingProduct.value.id, price_version_id: priceForm.price_version_id || undefined, billing_term: priceForm.billing_term, amount_fen: Math.round(priceForm.amount_yuan * 100), included_credits: priceForm.included_credits })
  priceVisible.value = false
  ElMessage.success('价格与额度已保存并发布')
  await load()
}

const valueOf = (version: PlatformPlanVersion | undefined, key: string) => {
  const value = version?.entitlements.find((item) => item.key === key)?.value_json
  return value ? Number(value) : 0
}

const openEditor = (plan: PlatformPlan, version?: PlatformPlanVersion) => {
  selectedPlan.value = plan
  selectedVersion.value = version || null
  form.version_id = version?.id || 0
  form.change_note = version?.change_note || ''
  form.members = valueOf(version, 'members.max') || 10
  form.jobs = valueOf(version, 'jobs.published.max') || 20
  form.applications = valueOf(version, 'applications.monthly.max') || 500
  form.resumes = valueOf(version, 'resumes.storage.max') || 1000
  form.aiHr = valueOf(version, 'ai.hr.enabled') !== 0
  form.aiChat = valueOf(version, 'ai.chat.enabled') !== 0
  form.aiResumeParse = valueOf(version, 'ai.resume_parse.enabled') !== 0
  form.aiMatchEvaluation = valueOf(version, 'ai.match_evaluation.enabled') !== 0
  form.aiApplicationAnalysis = valueOf(version, 'ai.application_analysis.enabled') !== 0
  form.aiAgentRun = valueOf(version, 'ai.agent_run.enabled') !== 0
  form.aiMonthlyCredits = valueOf(version, 'ai.credits.monthly') || 100
  form.aiConcurrentRuns = valueOf(version, 'ai.concurrent_runs.max') || 1
  form.aiSingleRunCredits = valueOf(version, 'ai.single_run.max_credits') || 20
  editorVisible.value = true
}

const entitlements = (): PlatformEntitlement[] => {
  const integers: Array<[string, number]> = [
    ['members.max', form.members], ['jobs.published.max', form.jobs],
    ['applications.monthly.max', form.applications], ['resumes.storage.max', form.resumes],
    ['ai.credits.monthly', form.aiMonthlyCredits], ['ai.concurrent_runs.max', form.aiConcurrentRuns],
    ['ai.single_run.max_credits', form.aiSingleRunCredits],
  ]
  const booleans: Array<[string, boolean]> = [
    ['ai.hr.enabled', form.aiHr], ['ai.chat.enabled', form.aiChat],
    ['ai.resume_parse.enabled', form.aiResumeParse], ['ai.match_evaluation.enabled', form.aiMatchEvaluation],
    ['ai.application_analysis.enabled', form.aiApplicationAnalysis], ['ai.agent_run.enabled', form.aiAgentRun],
  ]
  const releases: Array<[string, number]> = aiCapabilities.value
    .filter((item) => item.current_published_version_id > 0)
    .map((item) => [`${item.capability_key}.release_version_id`, item.current_published_version_id])
  return [
    ...integers.map(([key, value]) => ({ key, value_type: 'integer' as const, value_json: String(value), enforcement_mode: 'hard' as const })),
    ...booleans.map(([key, value]) => ({ key, value_type: 'boolean' as const, value_json: String(value), enforcement_mode: 'hard' as const })),
    ...releases.map(([key, value]) => ({ key, value_type: 'integer' as const, value_json: String(value), enforcement_mode: 'hard' as const })),
  ]
}

const displayValue = (item: PlatformEntitlement) => item.value_type === 'boolean'
  ? (item.value_json === 'true' ? '已启用' : '未启用')
  : Number(item.value_json).toLocaleString()

const submitDraft = async () => {
  if (!selectedPlan.value || !form.change_note.trim()) { ElMessage.warning('请填写版本变更说明'); return }
  await savePlanVersion(selectedPlan.value.id, { version_id: form.version_id || undefined, change_note: form.change_note.trim(), entitlements: entitlements() })
  editorVisible.value = false
  ElMessage.success('套餐版本草稿已保存')
  await load()
}

const openPublish = (plan: PlatformPlan, version: PlatformPlanVersion) => {
  selectedPlan.value = plan
  selectedVersion.value = version
  publishForm.effective_at = toShanghaiRFC3339(new Date())
  publishForm.reason = ''
  publishVisible.value = true
}

const submitPublish = async () => {
  if (!selectedPlan.value || !selectedVersion.value || !publishForm.reason.trim()) { ElMessage.warning('请填写发布原因'); return }
  await publishPlanVersion(selectedPlan.value.id, selectedVersion.value.id, { effective_at: toShanghaiRFC3339(publishForm.effective_at), reason: publishForm.reason.trim() })
  publishVisible.value = false
  ElMessage.success('套餐版本已发布')
  await load()
}

const formatTime = (value?: string) => formatShanghaiDateTime(value)
onMounted(load)
</script>

<template>
  <section class="console-page" v-loading="loading">
    <header class="catalog-heading"><div><span>COMMERCIAL CONTROL PLANE</span><h1>套餐与商业化</h1><p>分别维护平台权益、对外商品价格和模型成本换算，避免不同生命周期的配置混在同一工作区。</p></div></header>
    <el-tabs :model-value="activeSection" class="catalog-tabs" @tab-change="switchSection">
      <el-tab-pane label="套餐版本与权益" name="plans" />
      <el-tab-pane label="商品与价格" name="products" />
      <el-tab-pane label="AI 模型费率" name="rates" />
      <el-tab-pane v-if="canReviewRefund" label="退款审批" name="refunds" />
    </el-tabs>

    <div v-if="activeSection === 'plans'" class="plan-grid">
      <article v-for="plan in plans" :key="plan.id" class="surface-card plan-card">
        <header><div><span class="plan-key">{{ plan.plan_key }}</span><h2>{{ plan.name }}</h2><p>{{ plan.description }}</p></div><el-tag :type="plan.status === 'active' ? 'success' : 'info'">{{ plan.status === 'active' ? '启用' : '已退役' }}</el-tag></header>
        <div class="plan-version-list">
          <section v-for="version in plan.versions" :key="version.id" class="plan-version">
            <div class="plan-version__heading"><div><strong>版本 V{{ version.version }}</strong><small>{{ version.change_note || '暂无版本说明' }}</small></div><el-tag :type="version.status === 'published' ? 'success' : version.status === 'draft' ? 'warning' : 'info'" size="small">{{ version.status }}</el-tag></div>
            <div class="entitlement-grid"><div v-for="item in version.entitlements" :key="item.key"><span>{{ entitlementLabels[item.key] || item.key }}</span><strong>{{ displayValue(item) }}</strong><small>{{ item.enforcement_mode === 'hard' ? '硬限制' : item.enforcement_mode }}</small></div></div>
            <footer><span>{{ version.status === 'published' ? `生效：${formatTime(version.effective_at)}` : `更新：${formatTime(version.updated_at)}` }}</span><div><el-button v-if="canManage && version.status === 'draft'" link type="primary" @click="openEditor(plan, version)">编辑草稿</el-button><el-button v-if="canPublish && version.status === 'draft'" link type="success" @click="openPublish(plan, version)">发布</el-button></div></footer>
          </section>
        </div>
        <el-button v-if="canManage" class="plan-new-version" plain @click="openEditor(plan)">创建新版本草稿</el-button>
      </article>
    </div>

    <section v-if="activeSection === 'products'" class="surface-card billing-catalog"><header><div><h2>AI 计费商品与价格</h2><p>发布后会进入 HR 或候选人购买页；已发布价格不可修改，只能创建新版本。</p></div><el-tag type="warning">{{ paymentEnvironment === 'sandbox' ? '支付宝沙箱' : paymentEnvironment }}</el-tag></header><div class="billing-product-grid"><article v-for="product in billingProducts" :key="product.id"><div><strong>{{ product.name }}</strong><small>{{ product.product_key }} · {{ product.product_type }}</small></div><div v-if="product.prices[0]"><strong>¥{{ (product.prices[0].amount_fen / 100).toFixed(2) }}</strong><small>{{ product.prices[0].included_credits.toLocaleString() }} 额度 · V{{ product.prices[0].version }}</small></div><span v-else>尚未配置价格</span><el-button v-if="canManage && product.product_key !== 'candidate_free'" link type="primary" @click="openPriceEditor(product)">配置价格</el-button></article><el-empty v-if="!billingProducts.length" description="尚未创建计费商品" /></div></section>

    <section v-if="activeSection === 'rates'" class="surface-card billing-catalog"><header><div><h2>AI 模型费率卡</h2><p>供应商成本按每千 Token 配置；额度换算决定用户消耗，发布新版本后旧版本自动退役。</p></div><el-button v-if="canManage" type="primary" plain @click="openRateEditor()">新增费率卡</el-button></header><div class="billing-product-grid"><article v-for="rate in rateCards" :key="rate.id"><div><strong>{{ rate.provider_key }} / {{ rate.model_key }}</strong><small>V{{ rate.version }} · {{ rate.status }}</small></div><div><strong>输入 ¥{{ (rate.input_micros_per_1k_tokens / 1_000_000).toFixed(6) }}</strong><small>输出 ¥{{ (rate.output_micros_per_1k_tokens / 1_000_000).toFixed(6) }} / 千 Token</small></div><span>1 额度 = ¥{{ (rate.credit_micros / 1_000_000).toFixed(6) }}</span><el-button v-if="canManage" link type="primary" @click="openRateEditor(rate)">创建新版本</el-button></article><el-empty v-if="!rateCards.length" description="尚未配置模型费率，强制计费模式将拒绝未知模型" /></div></section>

    <section v-if="activeSection === 'refunds'" class="surface-card billing-catalog"><header><div><h2>退款审批与核对</h2><p>人工退款由平台资金责任人审批；未知状态由后台使用原退款号持续核对。</p></div></header><el-table :data="refunds" style="margin-top:20px"><el-table-column prop="refund_no" label="退款号" min-width="190"/><el-table-column prop="order_no" label="订单号" min-width="180"/><el-table-column label="金额" width="110"><template #default="{ row }">¥{{ (row.amount_fen / 100).toFixed(2) }}</template></el-table-column><el-table-column prop="reason" label="原因" min-width="180" show-overflow-tooltip/><el-table-column prop="status" label="状态" width="110"/><el-table-column label="操作" width="150"><template #default="{ row }"><template v-if="row.status === 'reviewing'"><el-button link type="success" @click="reviewRefund(row, 'approve')">批准</el-button><el-button link type="danger" @click="reviewRefund(row, 'reject')">拒绝</el-button></template><span v-else>{{ row.last_error || '—' }}</span></template></el-table-column></el-table><el-empty v-if="!refunds.length" description="暂无退款记录"/></section>

    <el-dialog v-model="editorVisible" :title="`${selectedPlan?.name || ''} · ${form.version_id ? '编辑草稿' : '新建版本'}`" width="680px">
      <el-alert title="已发布版本不可修改；保存新草稿不会立即影响任何租户。" type="info" :closable="false" show-icon />
      <el-form class="dialog-form" label-position="top"><el-form-item label="版本变更说明" required><el-input v-model="form.change_note" maxlength="500" show-word-limit placeholder="说明本版本权益调整背景" /></el-form-item><div class="two-columns"><el-form-item label="有效成员上限"><el-input-number v-model="form.members" :min="1" :max="1000000" controls-position="right" /></el-form-item><el-form-item label="在线岗位上限"><el-input-number v-model="form.jobs" :min="1" :max="1000000" controls-position="right" /></el-form-item><el-form-item label="月投递上限"><el-input-number v-model="form.applications" :min="1" :max="100000000" controls-position="right" /></el-form-item><el-form-item label="简历存储上限"><el-input-number v-model="form.resumes" :min="1" :max="100000000" controls-position="right" /></el-form-item></div><el-divider content-position="left">AI 权益</el-divider><div class="two-columns"><el-form-item label="HR AI 总开关"><el-switch v-model="form.aiHr" /></el-form-item><el-form-item label="AI 对话"><el-switch v-model="form.aiChat" /></el-form-item><el-form-item label="简历解析"><el-switch v-model="form.aiResumeParse" /></el-form-item><el-form-item label="匹配评估"><el-switch v-model="form.aiMatchEvaluation" /></el-form-item><el-form-item label="申请分析"><el-switch v-model="form.aiApplicationAnalysis" /></el-form-item><el-form-item label="Agent 任务"><el-switch v-model="form.aiAgentRun" /></el-form-item><el-form-item label="每月 AI 额度"><el-input-number v-model="form.aiMonthlyCredits" :min="1" :max="100000000" controls-position="right" /></el-form-item><el-form-item label="AI 并发任务"><el-input-number v-model="form.aiConcurrentRuns" :min="1" :max="1000" controls-position="right" /></el-form-item><el-form-item label="单次任务额度上限"><el-input-number v-model="form.aiSingleRunCredits" :min="1" :max="100000000" controls-position="right" /></el-form-item></div></el-form>
      <template #footer><el-button @click="editorVisible = false">取消</el-button><el-button type="primary" @click="submitDraft">保存草稿</el-button></template>
    </el-dialog>

    <el-dialog v-model="publishVisible" title="发布套餐版本" width="560px">
      <el-alert title="发布后版本内容将冻结，并可用于租户订阅。" type="warning" :closable="false" show-icon />
      <el-form class="dialog-form" label-position="top"><el-form-item label="生效时间" required><el-date-picker v-model="publishForm.effective_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" style="width:100%" /></el-form-item><el-form-item label="发布原因" required><el-input v-model="publishForm.reason" type="textarea" :rows="4" maxlength="500" show-word-limit /></el-form-item></el-form>
      <template #footer><el-button @click="publishVisible = false">取消</el-button><el-button type="primary" @click="submitPublish">确认发布</el-button></template>
    </el-dialog>

    <el-dialog v-model="priceVisible" :title="`${selectedBillingProduct?.name || ''} · 沙箱价格`" width="580px"><el-alert title="保存后会立即发布到对应购买页；当前只对接支付宝沙箱，不会计入真实营收。" type="warning" :closable="false" show-icon/><el-form class="dialog-form" label-position="top"><el-form-item label="计费周期"><el-select v-model="priceForm.billing_term" :disabled="selectedBillingProduct?.product_type === 'credit_pack'"><el-option label="按月" value="monthly"/><el-option label="按年" value="yearly"/><el-option label="一次性" value="one_time"/></el-select></el-form-item><div class="two-columns"><el-form-item label="沙箱价格（元）"><el-input-number v-model="priceForm.amount_yuan" :min="0.01" :precision="2" :step="1"/></el-form-item><el-form-item label="包含 AI 额度"><el-input-number v-model="priceForm.included_credits" :min="1" :step="100"/></el-form-item></div></el-form><template #footer><el-button @click="priceVisible=false">取消</el-button><el-button type="primary" @click="submitPrice">保存并发布</el-button></template></el-dialog>
    <el-dialog v-model="rateVisible" title="AI 模型费率卡新版本" width="620px"><el-alert title="供应商与模型来自已启用的 LLM 配置；保存后立即发布，旧版本自动退役。" type="info" :closable="false" show-icon/><el-form class="dialog-form" label-position="top"><div class="two-columns"><el-form-item label="供应商" required><el-select v-model="rateForm.provider_key" filterable placeholder="选择已启用供应商" :disabled="editingRateTarget" style="width:100%" @change="changeRateProvider"><el-option v-for="provider in rateProviderOptions" :key="provider.id" :label="provider.name" :value="provider.name"><span>{{ provider.name }}</span><small class="rate-option-meta">{{ provider.provider_type }}</small></el-option></el-select></el-form-item><el-form-item label="模型" required><el-select v-model="rateForm.model_key" filterable placeholder="选择该供应商下的已启用模型" :disabled="editingRateTarget || !rateForm.provider_key" style="width:100%"><el-option v-for="model in rateModelOptions" :key="model.id" :label="model.display_name || model.model_name" :value="model.model_name"><span>{{ model.display_name || model.model_name }}</span><small class="rate-option-meta">{{ model.model_name }}</small></el-option></el-select></el-form-item><el-form-item label="输入成本（元/千 Token）"><el-input-number v-model="rateForm.input_yuan" :min="0" :precision="6" :step="0.001"/></el-form-item><el-form-item label="输出成本（元/千 Token）"><el-input-number v-model="rateForm.output_yuan" :min="0" :precision="6" :step="0.001"/></el-form-item><el-form-item label="缓存输入成本（元/千 Token）"><el-input-number v-model="rateForm.cached_yuan" :min="0" :precision="6" :step="0.001"/></el-form-item><el-form-item label="每额度价值（元）" required><el-input-number v-model="rateForm.credit_yuan" :min="0.000001" :precision="6" :step="0.001"/></el-form-item></div></el-form><template #footer><el-button @click="rateVisible=false">取消</el-button><el-button type="primary" @click="submitRate">保存并发布</el-button></template></el-dialog>
  </section>
</template>

<style scoped>
.catalog-heading{margin-bottom:6px}.catalog-heading span{color:var(--el-color-primary);font-size:12px;letter-spacing:.12em}.catalog-heading h1{margin:6px 0;font-size:30px}.catalog-heading p{margin:0;color:var(--el-text-color-secondary)}.catalog-tabs{margin-bottom:18px}.billing-catalog{padding:24px}.billing-catalog>header{display:flex;justify-content:space-between;gap:20px;align-items:flex-start}.billing-catalog h2{margin:0 0 6px}.billing-catalog p{margin:0;color:var(--el-text-color-secondary)}.billing-product-grid{display:grid;gap:10px;margin-top:20px}.billing-product-grid article{display:grid;grid-template-columns:minmax(180px,1fr) minmax(160px,.7fr) minmax(110px,.5fr) auto;gap:16px;align-items:center;padding:14px 16px;border:1px solid var(--el-border-color-lighter);border-radius:12px}.billing-product-grid article>div{display:grid;gap:3px}.billing-product-grid small,.billing-product-grid span{color:var(--el-text-color-secondary)}.rate-option-meta{float:right;margin-left:16px;color:var(--el-text-color-secondary)}@media(max-width:760px){.billing-product-grid article{grid-template-columns:1fr}.billing-catalog>header{flex-direction:column}}</style>
