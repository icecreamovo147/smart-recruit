<script setup lang="ts">
import { t } from '@shared/i18n'
import { computed, onMounted, reactive, ref } from 'vue'
import { Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import { listQuotaAlerts, updateQuotaAlert } from '@/api/control'
import { DataTableCard, FilterToolbar, PageHeader, PagePanel } from '@/components/admin-console'
import { PLATFORM_PERMISSIONS } from '@/permissions'
import { useAuthStore } from '@/stores/auth'
import type { QuotaAlert } from '@/types'
import { formatShanghaiDateTime } from '@shared/utils/format'

const router = useRouter()
const auth = useAuthStore()
const canManage = computed(() => auth.can(PLATFORM_PERMISSIONS.ALERT_MANAGE))
const loading = ref(false)
const rows = ref<QuotaAlert[]>([])
const total = ref(0)
const query = reactive({ tenant_id: undefined as number | undefined, status: 'open', metric_key: '', page: 1, page_size: 20 })
const dialogVisible = ref(false)
const selected = ref<QuotaAlert | null>(null)
const action = reactive({ status: 'acknowledged' as 'acknowledged' | 'resolved', resolution_note: '' })
const metricLabels: Record<string, string> = { 'members.max': '有效成员', 'jobs.published.max': '在线岗位', 'applications.monthly.max': '月投递量', 'resumes.storage.max': '简历存储' }
const statusLabel = (status: QuotaAlert['status']) => ({ open: '待处理', acknowledged: '已认领', resolved: '已解决' }[status])

const load = async () => {
  loading.value = true
  try { const response = await listQuotaAlerts(query); rows.value = response.list || []; total.value = Number(response.total) || 0 } finally { loading.value = false }
}
const search = () => { query.page = 1; load() }
const openAction = (row: QuotaAlert, status: 'acknowledged' | 'resolved') => { selected.value = row; action.status = status; action.resolution_note = ''; dialogVisible.value = true }
const submit = async () => {
  if (!selected.value) return
  if (action.status === 'resolved' && !action.resolution_note.trim()) { ElMessage.warning(t('common.invalid_request')); return }
  await updateQuotaAlert(selected.value.id, { status: action.status, assignee_user_id: auth.user?.user_id, resolution_note: action.resolution_note.trim() })
  dialogVisible.value = false
  ElMessage.success(action.status === 'resolved' ? '告警已解决' : '告警已认领')
  await load()
}
const formatTime = (value?: string) => formatShanghaiDateTime(value)
onMounted(load)
</script>

<template>
  <section class="console-page" v-loading="loading">
    <PagePanel>
      <PageHeader title="配额告警" kicker="QUOTA ALERTS" description="跟踪租户配额阈值触达与处置状态" />

      <FilterToolbar>
        <div class="filter-fields filter-fields--wrap">
          <el-input-number v-model="query.tenant_id" :min="1" :controls="false" placeholder="租户 ID" />
          <el-select v-model="query.status" clearable placeholder="告警状态"><el-option label="待处理" value="open" /><el-option label="已认领" value="acknowledged" /><el-option label="已解决" value="resolved" /></el-select>
          <el-select v-model="query.metric_key" clearable placeholder="配额指标"><el-option v-for="(label, key) in metricLabels" :key="key" :label="label" :value="key" /></el-select>
        </div>
        <template #actions>
          <div class="filter-actions"><el-button type="primary" :icon="Search" @click="search">查询</el-button></div>
        </template>
      </FilterToolbar>

      <DataTableCard :result-count="total" :result-label="`共 ${total} 条告警`">
        <el-table :data="rows" stripe class="console-table" @row-click="(row: QuotaAlert) => router.push(`/tenants/${row.tenant_id}`)">
          <el-table-column label="租户" min-width="210"><template #default="{ row }"><div class="tenant-cell"><span class="tenant-avatar">{{ row.tenant_name?.slice(0, 1) || '-' }}</span><div><strong>{{ row.tenant_name || `租户 ${row.tenant_id}` }}</strong><small>ID {{ row.tenant_id }}</small></div></div></template></el-table-column>
          <el-table-column label="指标" min-width="150"><template #default="{ row }"><strong>{{ metricLabels[row.metric_key] || row.metric_key }}</strong></template></el-table-column>
          <el-table-column label="当前用量" width="190"><template #default="{ row }"><div class="quota-progress"><span>{{ row.usage_value.toLocaleString() }} / {{ row.quota_value.toLocaleString() }}</span><el-progress :percentage="Math.min(100, Math.round(row.usage_value * 100 / row.quota_value))" :status="row.threshold_percent >= 100 ? 'exception' : 'warning'" :show-text="false" /></div></template></el-table-column>
          <el-table-column label="阈值" width="90"><template #default="{ row }"><el-tag :type="row.threshold_percent >= 100 ? 'danger' : 'warning'">{{ row.threshold_percent }}%</el-tag></template></el-table-column>
          <el-table-column label="状态" width="110"><template #default="{ row }"><el-tag :type="row.status === 'resolved' ? 'success' : row.status === 'acknowledged' ? 'primary' : 'warning'">{{ statusLabel(row.status) }}</el-tag></template></el-table-column>
          <el-table-column label="最近触发" width="180"><template #default="{ row }">{{ formatTime(row.last_triggered_at) }}</template></el-table-column>
          <el-table-column v-if="canManage" label="操作" width="150" align="center"><template #default="{ row }"><el-button v-if="row.status === 'open'" link type="primary" @click.stop="openAction(row, 'acknowledged')">认领</el-button><el-button v-if="row.status !== 'resolved'" link type="success" @click.stop="openAction(row, 'resolved')">解决</el-button></template></el-table-column>
        </el-table>
        <template #footer>
          <el-pagination v-model:current-page="query.page" v-model:page-size="query.page_size" :total="total" layout="prev, pager, next, sizes" :page-sizes="[10,20,50,100]" @current-change="load" @size-change="load" />
        </template>
      </DataTableCard>
    </PagePanel>

    <el-dialog v-model="dialogVisible" :title="action.status === 'resolved' ? '解决配额告警' : '认领配额告警'" width="520px"><el-form label-position="top"><el-form-item label="处置说明" :required="action.status === 'resolved'"><el-input v-model="action.resolution_note" type="textarea" :rows="4" maxlength="500" show-word-limit placeholder="记录沟通情况、扩容方案或问题结论" /></el-form-item></el-form><template #footer><el-button @click="dialogVisible = false">取消</el-button><el-button type="primary" @click="submit">确认</el-button></template></el-dialog>
  </section>
</template>
