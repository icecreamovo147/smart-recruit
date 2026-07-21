<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { DocumentCopy, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useRoute } from 'vue-router'
import { listPlatformAuditLogs } from '@/api/audit'
import type { PlatformAuditLog } from '@/types'
import { formatShanghaiDateTime, toShanghaiRFC3339 } from '@shared/utils/format'

const route = useRoute()
const loading = ref(false)
const logs = ref<PlatformAuditLog[]>([])
const total = ref(0)
const selected = ref<PlatformAuditLog | null>(null)
const detailVisible = ref(false)
const dateRange = ref<[Date, Date] | null>(null)
const query = reactive({
  page: 1, page_size: 20,
  tenant_id: Number(route.query.tenant_id) || undefined as number | undefined,
  actor_user_id: undefined as number | undefined,
  action: '', request_id: '',
})

const load = async () => {
  loading.value = true
  try {
    const response = await listPlatformAuditLogs({
      ...query,
      tenant_id: query.tenant_id || undefined,
      actor_user_id: query.actor_user_id || undefined,
      action: query.action || undefined,
      request_id: query.request_id.trim() || undefined,
      start_time: dateRange.value ? toShanghaiRFC3339(dateRange.value[0]) : undefined,
      end_time: dateRange.value ? toShanghaiRFC3339(dateRange.value[1]) : undefined,
    })
    logs.value = response.list || []
    total.value = Number(response.total) || 0
  } finally { loading.value = false }
}

const search = () => { query.page = 1; void load() }
const reset = () => {
  Object.assign(query, { page: 1, tenant_id: undefined, actor_user_id: undefined, action: '', request_id: '' })
  dateRange.value = null
  void load()
}

const openDetail = (row: PlatformAuditLog) => { selected.value = row; detailVisible.value = true }
const formatTime = (value: string) => formatShanghaiDateTime(value)
const actionLabels: Record<string, string> = {
  'tenant.create': '创建租户',
  'tenant.status.update': '租户状态变更',
  'tenant.membership.status.update': '成员状态变更',
  'platform_user.create': '创建平台账号',
  'platform_user.update': '变更平台账号',
  'plan.version.save': '保存套餐版本',
  'plan.version.publish': '发布套餐版本',
  'tenant.subscription.update': '变更租户套餐',
  'tenant.entitlement.override': '设置租户权益覆盖',
  'quota_alert.status.update': '更新配额告警',
}
const actionLabel = (action: string) => actionLabels[action] || action
const prettyJson = (value: string) => {
  if (!value) return '无'
  try { return JSON.stringify(JSON.parse(value), null, 2) } catch { return value }
}
const beforeText = computed(() => prettyJson(selected.value?.before_json || ''))
const afterText = computed(() => prettyJson(selected.value?.after_json || ''))
const copyRequestId = async (value: string) => {
  if (!value) return
  await navigator.clipboard.writeText(value)
  ElMessage.success('Request ID 已复制')
}

onMounted(load)
</script>

<template>
  <section class="console-page">
    <article class="surface-card table-surface">
      <div class="filter-toolbar filter-toolbar--wrap">
        <div class="filter-fields filter-fields--wrap">
          <el-input v-model.number="query.tenant_id" placeholder="租户 ID" clearable />
          <el-input v-model.number="query.actor_user_id" placeholder="操作人 ID" clearable />
          <el-select v-model="query.action" clearable placeholder="全部操作"><el-option v-for="(label, value) in actionLabels" :key="value" :label="label" :value="value" /></el-select>
          <el-input v-model="query.request_id" placeholder="Request ID" clearable />
          <el-date-picker v-model="dateRange" class="audit-date-range" type="datetimerange" range-separator="至" start-placeholder="开始时间" end-placeholder="结束时间" />
        </div>
        <div class="filter-actions"><el-button type="primary" :icon="Search" @click="search">查询</el-button><el-button @click="reset">重置</el-button></div>
      </div>
      <el-table v-loading="loading" :data="logs" stripe class="console-table" @row-click="openDetail">
        <el-table-column label="时间" width="180"><template #default="{ row }">{{ formatTime(row.created_at) }}</template></el-table-column>
        <el-table-column label="操作" min-width="170"><template #default="{ row }"><strong>{{ actionLabel(row.action) }}</strong><small class="cell-secondary">{{ row.resource_type }} #{{ row.resource_id }}</small></template></el-table-column>
        <el-table-column label="目标企业" min-width="180"><template #default="{ row }">{{ row.target_tenant_name || '-' }}<small class="cell-secondary">Tenant ID {{ row.target_tenant_id || '-' }}</small></template></el-table-column>
        <el-table-column label="操作人" min-width="150"><template #default="{ row }">{{ row.actor_username || '-' }}<small class="cell-secondary">用户 ID {{ row.actor_user_id }}</small></template></el-table-column>
        <el-table-column prop="client_ip" label="客户端 IP" width="150" />
        <el-table-column label="Request ID" min-width="190"><template #default="{ row }"><button v-if="row.request_id" class="request-id" type="button" @click.stop="copyRequestId(row.request_id)">{{ row.request_id }}<el-icon><DocumentCopy /></el-icon></button><span v-else>-</span></template></el-table-column>
        <el-table-column label="详情" width="80" fixed="right"><template #default="{ row }"><el-button link type="primary" @click.stop="openDetail(row)">查看</el-button></template></el-table-column>
      </el-table>
      <footer class="table-footer"><span>共 {{ total }} 条操作记录</span><el-pagination v-model:current-page="query.page" v-model:page-size="query.page_size" :total="total" layout="prev, pager, next, sizes" :page-sizes="[10, 20, 50, 100]" @current-change="load" @size-change="search" /></footer>
    </article>

    <el-drawer v-model="detailVisible" title="操作审计详情" size="680px">
      <div v-if="selected" class="audit-detail">
        <dl class="description-list"><div><dt>操作</dt><dd>{{ actionLabel(selected.action) }}</dd></div><div><dt>操作人</dt><dd>{{ selected.actor_username || '-' }}（{{ selected.actor_user_id }}）</dd></div><div><dt>目标企业</dt><dd>{{ selected.target_tenant_name || '-' }}（{{ selected.target_tenant_id }}）</dd></div><div><dt>发生时间</dt><dd>{{ formatTime(selected.created_at) }}</dd></div><div><dt>客户端 IP</dt><dd>{{ selected.client_ip || '-' }}</dd></div><div><dt>Request ID</dt><dd class="mono">{{ selected.request_id || '-' }}</dd></div></dl>
        <section class="json-comparison"><div><h3>变更前</h3><pre>{{ beforeText }}</pre></div><div><h3>变更后</h3><pre>{{ afterText }}</pre></div></section>
      </div>
    </el-drawer>
  </section>
</template>
