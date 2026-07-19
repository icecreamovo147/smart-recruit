<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ArrowLeft, Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { listPlatformAuditLogs } from '@/api/audit'
import { getTenantSubscription, getTenantUsage, listPlans, updateTenantEntitlementOverride, updateTenantSubscription } from '@/api/control'
import { getTenant, listMemberships, updateMembershipStatus, updateTenantStatus } from '@/api/tenant'
import { PLATFORM_PERMISSIONS, roleLabel } from '@/permissions'
import { useAuthStore } from '@/stores/auth'
import type { Membership, PlatformAuditLog, PlatformPlan, Tenant, TenantSubscription, TenantUsageMetric } from '@/types'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const tenantId = Number(route.params.tenantId)
const canManageTenant = computed(() => auth.can(PLATFORM_PERMISSIONS.TENANT_MANAGE))
const canManageMembers = computed(() => auth.can(PLATFORM_PERMISSIONS.MEMBER_MANAGE))
const canReadAudit = computed(() => auth.can(PLATFORM_PERMISSIONS.AUDIT_READ))
const canReadUsage = computed(() => auth.can(PLATFORM_PERMISSIONS.USAGE_READ))
const canManageSubscription = computed(() => auth.can(PLATFORM_PERMISSIONS.SUBSCRIPTION_MANAGE))
const canManagePlan = computed(() => auth.can(PLATFORM_PERMISSIONS.PLAN_MANAGE))
const loading = ref(false)
const tenant = ref<Tenant | null>(null)
const memberships = ref<Membership[]>([])
const membershipTotal = ref(0)
const auditLogs = ref<PlatformAuditLog[]>([])
const subscription = ref<TenantSubscription | null>(null)
const usageMetrics = ref<TenantUsageMetric[]>([])
const plans = ref<PlatformPlan[]>([])
const activeTab = ref('overview')
const memberQuery = reactive({ page: 1, page_size: 20 })
const statusVisible = ref(false)
const statusTarget = ref<{ kind: 'tenant' | 'membership'; status: string; membership?: Membership } | null>(null)
const reason = ref('')
const subscriptionVisible = ref(false)
const subscriptionForm = reactive({ plan_version_id: 0, starts_at: '', ends_at: '', reason: '' })
const overrideVisible = ref(false)
const overrideForm = reactive({ entitlement_key: '', quota_value: 1, expires_at: '', reason: '' })

const metricLabels: Record<string, string> = { 'members.max': '有效成员', 'jobs.published.max': '在线岗位', 'applications.monthly.max': '月投递量', 'resumes.storage.max': '简历存储' }
const publishedVersions = computed(() => plans.value.flatMap((plan) => plan.versions.filter((version) => version.status === 'published').map((version) => ({ ...version, planName: plan.name }))))

const loadTenant = async () => {
  const response = await getTenant(tenantId)
  tenant.value = response.tenant
}

const loadMembers = async () => {
  const response = await listMemberships(tenantId, memberQuery)
  memberships.value = response.list || []
  membershipTotal.value = Number(response.total) || 0
}

const loadAudit = async () => {
  if (!canReadAudit.value) return
  const response = await listPlatformAuditLogs({ tenant_id: tenantId, page: 1, page_size: 50 })
  auditLogs.value = response.list || []
}

const loadCommercial = async () => {
  const jobs: Promise<unknown>[] = [getTenantSubscription(tenantId).then((response) => { subscription.value = response.subscription || null })]
  if (canReadUsage.value) jobs.push(getTenantUsage(tenantId).then((response) => { usageMetrics.value = response.metrics || [] }))
  if (canManageSubscription.value) jobs.push(listPlans('active').then((response) => { plans.value = response.list || [] }))
  await Promise.all(jobs)
}

const load = async () => {
  if (!Number.isInteger(tenantId) || tenantId <= 0) { await router.replace('/tenants'); return }
  loading.value = true
  try { await Promise.all([loadTenant(), loadMembers(), loadAudit(), loadCommercial()]) } finally { loading.value = false }
}

const openTenantStatus = (status: Tenant['status']) => {
  statusTarget.value = { kind: 'tenant', status }
  reason.value = ''
  statusVisible.value = true
}

const openMemberStatus = (membership: Membership, status: Membership['membership_status']) => {
  statusTarget.value = { kind: 'membership', status, membership }
  reason.value = ''
  statusVisible.value = true
}

const submitStatus = async () => {
  if (!statusTarget.value || !reason.value.trim()) { ElMessage.warning('请填写变更原因'); return }
  if (statusTarget.value.kind === 'tenant') {
    await updateTenantStatus(tenantId, statusTarget.value.status, reason.value.trim())
    await loadTenant()
  } else if (statusTarget.value.membership) {
    await updateMembershipStatus(tenantId, statusTarget.value.membership.membership_id, statusTarget.value.status, reason.value.trim())
    await loadMembers()
  }
  await loadAudit()
  statusVisible.value = false
  ElMessage.success('状态已更新')
}

const statusMeta = (status: string) => ({
  active: { label: '正常', type: 'success' }, suspended: { label: '已暂停', type: 'warning' }, disabled: { label: '已停用', type: 'info' },
}[status] || { label: status, type: 'info' })

const actionLabel = (action: string) => ({
  'tenant.create': '创建租户', 'tenant.status.update': '变更租户状态', 'tenant.membership.status.update': '变更成员状态',
  'tenant.subscription.update': '变更租户套餐', 'tenant.entitlement.override': '设置租户权益覆盖',
  'quota_alert.status.update': '更新配额告警',
}[action] || action)

const formatTime = (value?: string) => value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'

const openSubscription = () => {
  subscriptionForm.plan_version_id = subscription.value?.plan_version_id || 0
  subscriptionForm.starts_at = new Date().toISOString()
  subscriptionForm.ends_at = ''
  subscriptionForm.reason = ''
  subscriptionVisible.value = true
}

const submitSubscription = async () => {
  if (!subscriptionForm.plan_version_id || !subscriptionForm.reason.trim()) { ElMessage.warning('请选择套餐版本并填写变更原因'); return }
  if (new Date(subscriptionForm.starts_at).getTime() > Date.now()) { ElMessage.warning('第一、二阶段仅支持立即生效或回溯生效，请勿选择未来时间'); return }
  await updateTenantSubscription(tenantId, { plan_version_id: subscriptionForm.plan_version_id, starts_at: new Date(subscriptionForm.starts_at).toISOString(), ends_at: subscriptionForm.ends_at ? new Date(subscriptionForm.ends_at).toISOString() : undefined, reason: subscriptionForm.reason.trim() })
  subscriptionVisible.value = false
  ElMessage.success('租户订阅已更新')
  await Promise.all([loadCommercial(), loadAudit()])
}

const openOverride = (metric: TenantUsageMetric) => {
  overrideForm.entitlement_key = metric.key
  overrideForm.quota_value = metric.quota_value
  overrideForm.expires_at = ''
  overrideForm.reason = ''
  overrideVisible.value = true
}

const submitOverride = async () => {
  if (!overrideForm.entitlement_key || !Number.isInteger(overrideForm.quota_value) || overrideForm.quota_value <= 0 || !overrideForm.reason.trim()) {
    ElMessage.warning('请填写有效的正整数配额和变更原因')
    return
  }
  await updateTenantEntitlementOverride(tenantId, {
    entitlement_key: overrideForm.entitlement_key,
    value_type: 'integer',
    value_json: String(overrideForm.quota_value),
    expires_at: overrideForm.expires_at ? new Date(overrideForm.expires_at).toISOString() : undefined,
    reason: overrideForm.reason.trim(),
  })
  overrideVisible.value = false
  ElMessage.success('租户专属配额已更新')
  await Promise.all([loadCommercial(), loadAudit()])
}

onMounted(load)
</script>

<template>
  <section class="console-page tenant-detail" v-loading="loading">
    <button class="back-link" type="button" @click="router.push('/tenants')"><el-icon><ArrowLeft /></el-icon>返回租户列表</button>
    <header class="detail-hero">
      <div class="detail-identity"><span class="tenant-avatar tenant-avatar--large">{{ tenant?.name?.slice(0, 1) || '-' }}</span><div><div class="detail-title-row"><h1>{{ tenant?.name || '租户详情' }}</h1><el-tag v-if="tenant" :type="statusMeta(tenant.status).type">{{ statusMeta(tenant.status).label }}</el-tag><el-tag v-if="tenant?.is_default" type="info">默认租户</el-tag></div><p>{{ tenant?.slug }} · {{ tenant?.tenant_key }}</p></div></div>
      <div class="page-actions"><el-button :icon="Refresh" @click="load">刷新</el-button><template v-if="canManageTenant && tenant && !tenant.is_default"><el-button v-if="tenant.status !== 'suspended'" type="warning" plain @click="openTenantStatus('suspended')">暂停租户</el-button><el-button v-if="tenant.status !== 'active'" type="success" plain @click="openTenantStatus('active')">恢复租户</el-button></template></div>
    </header>

    <article class="surface-card detail-tabs">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="概览" name="overview">
          <div class="overview-grid">
            <section><h2>企业身份</h2><dl class="description-list"><div><dt>企业名称</dt><dd>{{ tenant?.name || '-' }}</dd></div><div><dt>企业标识</dt><dd>{{ tenant?.slug || '-' }}</dd></div><div><dt>Tenant Key</dt><dd class="mono">{{ tenant?.tenant_key || '-' }}</dd></div><div><dt>生命周期状态</dt><dd>{{ statusMeta(tenant?.status || '').label }}</dd></div></dl></section>
            <section><h2>本地化与记录</h2><dl class="description-list"><div><dt>时区</dt><dd>{{ tenant?.timezone || '-' }}</dd></div><div><dt>语言</dt><dd>{{ tenant?.locale || '-' }}</dd></div><div><dt>创建时间</dt><dd>{{ formatTime(tenant?.created_at) }}</dd></div><div><dt>更新时间</dt><dd>{{ formatTime(tenant?.updated_at) }}</dd></div></dl></section>
          </div>
          <div class="summary-strip"><div><span>成员关系</span><strong>{{ membershipTotal }}</strong><small>包含正常与暂停成员</small></div><div><span>有效成员</span><strong>{{ memberships.filter((item) => item.membership_status === 'active').length }}</strong><small>当前页内有效成员</small></div><div><span>招聘管理员</span><strong>{{ memberships.filter((item) => item.membership_status === 'active' && item.roles.includes('recruiting_admin')).length }}</strong><small>当前页内有效管理员</small></div></div>
        </el-tab-pane>

        <el-tab-pane :label="`成员 (${membershipTotal})`" name="members">
          <div class="tab-heading"><div><h2>企业成员</h2><p>平台仅处理准入和紧急状态治理，企业内部业务权限仍由租户管理员负责。</p></div></div>
          <el-table :data="memberships" stripe class="console-table">
            <el-table-column label="成员" min-width="220"><template #default="{ row }"><div class="user-cell"><span>{{ row.username.slice(0, 1).toUpperCase() }}</span><div><strong>{{ row.username }}</strong><small>{{ row.email || `用户 ID ${row.user_id}` }}</small></div></div></template></el-table-column>
            <el-table-column label="租户角色" min-width="200"><template #default="{ row }"><div class="tag-list"><el-tag v-for="role in row.roles" :key="role" type="info" size="small">{{ roleLabel(role) }}</el-tag><span v-if="!row.roles.length">-</span></div></template></el-table-column>
            <el-table-column label="状态" width="110"><template #default="{ row }"><el-tag :type="statusMeta(row.membership_status).type">{{ statusMeta(row.membership_status).label }}</el-tag></template></el-table-column>
            <el-table-column label="加入时间" width="180"><template #default="{ row }">{{ formatTime(row.joined_at) }}</template></el-table-column>
            <el-table-column v-if="canManageMembers" label="操作" width="120" align="center"><template #default="{ row }"><el-button v-if="row.membership_status === 'active'" link type="warning" @click="openMemberStatus(row, 'suspended')">暂停</el-button><el-button v-else link type="success" @click="openMemberStatus(row, 'active')">恢复</el-button></template></el-table-column>
          </el-table>
          <footer class="table-footer"><span>共 {{ membershipTotal }} 名成员</span><el-pagination v-model:current-page="memberQuery.page" v-model:page-size="memberQuery.page_size" :total="membershipTotal" layout="prev, pager, next, sizes" :page-sizes="[10, 20, 50, 100]" @current-change="loadMembers" @size-change="loadMembers" /></footer>
        </el-tab-pane>

        <el-tab-pane label="套餐与用量" name="subscription">
          <div class="tab-heading"><div><h2>订阅与配额</h2><p>查看当前生效权益、实时用量和阈值风险。</p></div><el-button v-if="canManageSubscription" type="primary" @click="openSubscription">变更套餐</el-button></div>
          <el-empty v-if="!subscription" :image-size="72" description="当前租户尚未配置套餐"><el-button v-if="canManageSubscription" type="primary" @click="openSubscription">配置首个套餐</el-button></el-empty>
          <template v-else>
            <div class="subscription-summary"><div><span>当前套餐</span><strong>{{ subscription.plan_name }} <small>V{{ subscription.plan_version }}</small></strong></div><div><span>订阅状态</span><el-tag type="success">{{ subscription.status }}</el-tag></div><div><span>开始时间</span><strong>{{ formatTime(subscription.starts_at) }}</strong></div><div><span>结束时间</span><strong>{{ formatTime(subscription.ends_at) }}</strong></div></div>
            <div v-if="canReadUsage" class="usage-grid"><article v-for="metric in usageMetrics" :key="metric.key" class="usage-card"><div><span>{{ metricLabels[metric.key] || metric.key }}</span><el-tag size="small" :type="metric.usage_percent >= 100 ? 'danger' : metric.usage_percent >= 80 ? 'warning' : 'success'">{{ metric.enforcement_mode === 'hard' ? '硬限制' : metric.enforcement_mode }}</el-tag></div><strong>{{ metric.usage_value.toLocaleString() }} <small>/ {{ metric.quota_value.toLocaleString() }}</small></strong><el-progress :percentage="Math.min(100, metric.usage_percent)" :status="metric.usage_percent >= 100 ? 'exception' : metric.usage_percent >= 80 ? 'warning' : 'success'" /><footer><p>采集于 {{ formatTime(metric.measured_at) }}</p><el-button v-if="canManagePlan" link type="primary" @click="openOverride(metric)">设置租户覆盖</el-button></footer></article></div>
          </template>
        </el-tab-pane>

        <el-tab-pane v-if="canReadAudit" label="操作记录" name="audit">
          <div class="tab-heading"><div><h2>租户操作时间线</h2><p>记录平台侧对该租户执行的生命周期和成员治理操作。</p></div><el-button link type="primary" @click="router.push({ path: '/audit-logs', query: { tenant_id: tenantId } })">查看完整审计</el-button></div>
          <el-timeline class="audit-timeline"><el-timeline-item v-for="item in auditLogs" :key="item.id" :timestamp="formatTime(item.created_at)" placement="top"><div class="timeline-card"><strong>{{ actionLabel(item.action) }}</strong><p>{{ item.actor_username || `用户 ${item.actor_user_id}` }} · {{ item.client_ip || '未知 IP' }}</p><code v-if="item.request_id">{{ item.request_id }}</code></div></el-timeline-item></el-timeline>
          <el-empty v-if="!auditLogs.length" :image-size="72" description="暂无平台操作记录" />
        </el-tab-pane>
      </el-tabs>
    </article>

    <el-dialog v-model="statusVisible" title="确认状态变更" width="520px">
      <el-alert type="warning" :title="statusTarget?.kind === 'tenant' ? '租户状态变更会影响企业整体访问' : '成员状态变更会立即影响该账号访问'" :closable="false" show-icon />
      <el-form class="dialog-form" label-position="top"><el-form-item label="变更原因" required><el-input v-model="reason" type="textarea" :rows="4" maxlength="500" show-word-limit placeholder="填写业务背景、恢复条件或审批依据" /></el-form-item></el-form>
      <template #footer><el-button @click="statusVisible = false">取消</el-button><el-button type="primary" @click="submitStatus">确认变更</el-button></template>
    </el-dialog>
    <el-dialog v-model="subscriptionVisible" title="变更租户套餐" width="600px"><el-alert title="套餐变更会保留历史订阅记录；第一、二阶段支持立即或回溯生效。" type="warning" :closable="false" show-icon /><el-form class="dialog-form" label-position="top"><el-form-item label="套餐版本" required><el-select v-model="subscriptionForm.plan_version_id" style="width:100%" placeholder="选择已发布版本"><el-option v-for="version in publishedVersions" :key="version.id" :label="`${version.planName} V${version.version}`" :value="version.id" /></el-select></el-form-item><div class="two-columns"><el-form-item label="开始时间" required><el-date-picker v-model="subscriptionForm.starts_at" type="datetime" :disabled-date="(date: Date) => date.getTime() > Date.now()" style="width:100%" /></el-form-item><el-form-item label="结束时间"><el-date-picker v-model="subscriptionForm.ends_at" type="datetime" style="width:100%" /></el-form-item></div><el-form-item label="变更原因" required><el-input v-model="subscriptionForm.reason" type="textarea" :rows="4" maxlength="500" show-word-limit /></el-form-item></el-form><template #footer><el-button @click="subscriptionVisible = false">取消</el-button><el-button type="primary" @click="submitSubscription">确认变更</el-button></template></el-dialog>
    <el-dialog v-model="overrideVisible" title="设置租户专属配额" width="560px"><el-alert title="专属配额会覆盖当前套餐中的同名权益；到期后自动恢复套餐值。" type="warning" :closable="false" show-icon /><el-form class="dialog-form" label-position="top"><el-form-item label="权益项"><el-input :model-value="metricLabels[overrideForm.entitlement_key] || overrideForm.entitlement_key" disabled /></el-form-item><div class="two-columns"><el-form-item label="配额值" required><el-input-number v-model="overrideForm.quota_value" :min="1" :step="1" step-strictly controls-position="right" style="width:100%" /></el-form-item><el-form-item label="失效时间"><el-date-picker v-model="overrideForm.expires_at" type="datetime" :disabled-date="(date: Date) => date.getTime() < Date.now() - 86400000" style="width:100%" placeholder="不填则长期生效" /></el-form-item></div><el-form-item label="变更原因" required><el-input v-model="overrideForm.reason" type="textarea" :rows="4" maxlength="500" show-word-limit placeholder="填写审批依据、客户需求或临时扩容背景" /></el-form-item></el-form><template #footer><el-button @click="overrideVisible = false">取消</el-button><el-button type="primary" @click="submitOverride">确认覆盖</el-button></template></el-dialog>
  </section>
</template>
