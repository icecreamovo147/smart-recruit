<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { MoreFilled, Plus, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { createTenant, listTenants, updateTenantStatus } from '@/api/tenant'
import { PLATFORM_PERMISSIONS } from '@/permissions'
import { useAuthStore } from '@/stores/auth'
import type { Tenant } from '@/types'
import { formatShanghaiDateTime } from '@shared/utils/format'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const canManage = computed(() => auth.can(PLATFORM_PERMISSIONS.TENANT_MANAGE))
const loading = ref(false)
const creating = ref(false)
const createVisible = ref(false)
const statusVisible = ref(false)
const tenants = ref<Tenant[]>([])
const total = ref(0)
const selectedTenant = ref<Tenant | null>(null)
const targetStatus = ref<Tenant['status']>('active')
const statusReason = ref('')
const query = reactive({ page: 1, page_size: 20, keyword: '', status: String(route.query.status || '') })
const form = reactive({ slug: '', name: '', timezone: 'Asia/Shanghai', locale: 'zh-CN' })

const load = async () => {
  loading.value = true
  try {
    const response = await listTenants(query)
    tenants.value = response.list || []
    total.value = Number(response.total) || 0
  } finally {
    loading.value = false
  }
}

const search = () => { query.page = 1; void load() }
const reset = () => { Object.assign(query, { page: 1, keyword: '', status: '' }); void load() }

const submitCreate = async () => {
  if (!form.slug.trim() || !form.name.trim()) { ElMessage.warning('请填写企业标识与名称'); return }
  creating.value = true
  try {
    const response = await createTenant({ ...form, slug: form.slug.trim(), name: form.name.trim() })
    createVisible.value = false
    Object.assign(form, { slug: '', name: '', timezone: 'Asia/Shanghai', locale: 'zh-CN' })
    ElMessage.success('企业租户已创建')
    await router.push(`/tenants/${response.tenant.id}`)
  } finally { creating.value = false }
}

const openStatusChange = (tenant: Tenant, status: Tenant['status']) => {
  selectedTenant.value = tenant
  targetStatus.value = status
  statusReason.value = ''
  statusVisible.value = true
}

const submitStatusChange = async () => {
  if (!selectedTenant.value || !statusReason.value.trim()) { ElMessage.warning('请填写状态变更原因'); return }
  await updateTenantStatus(selectedTenant.value.id, targetStatus.value, statusReason.value.trim())
  statusVisible.value = false
  ElMessage.success('租户状态已更新')
  await load()
}

const statusMeta = (status: string) => ({
  active: { label: '正常', type: 'success' }, suspended: { label: '已暂停', type: 'warning' }, disabled: { label: '已停用', type: 'info' },
}[status] || { label: status, type: 'info' })

const handleCommand = (tenant: Tenant, command: string) => {
  if (command === 'detail') void router.push(`/tenants/${tenant.id}`)
  if (command === 'suspend') openStatusChange(tenant, 'suspended')
  if (command === 'activate') openStatusChange(tenant, 'active')
  if (command === 'disable') openStatusChange(tenant, 'disabled')
}

onMounted(load)
</script>

<template>
  <section class="console-page">
    <article class="surface-card table-surface">
      <div class="filter-toolbar">
        <div class="filter-fields">
          <el-input v-model="query.keyword" :prefix-icon="Search" clearable placeholder="搜索企业名称、标识或 Tenant Key" @keyup.enter="search" />
          <el-select v-model="query.status" clearable placeholder="全部状态"><el-option label="正常" value="active" /><el-option label="已暂停" value="suspended" /><el-option label="已停用" value="disabled" /></el-select>
        </div>
        <div class="filter-actions"><el-button type="primary" @click="search">查询</el-button><el-button @click="reset">重置</el-button><el-button v-if="canManage" type="primary" :icon="Plus" @click="createVisible = true">创建企业</el-button></div>
      </div>
      <el-table v-loading="loading" :data="tenants" class="console-table" stripe @row-click="(row: Tenant) => router.push(`/tenants/${row.id}`)">
        <el-table-column label="企业" min-width="250"><template #default="{ row }"><div class="tenant-cell"><span class="tenant-avatar">{{ row.name.slice(0, 1) }}</span><div><strong>{{ row.name }}</strong><small>{{ row.slug }} · {{ row.tenant_key }}</small></div></div></template></el-table-column>
        <el-table-column label="状态" width="110"><template #default="{ row }"><el-tag :type="statusMeta(row.status).type">{{ statusMeta(row.status).label }}</el-tag></template></el-table-column>
        <el-table-column prop="membership_count" label="成员数" width="100" align="right" />
        <el-table-column prop="timezone" label="时区" min-width="150" />
        <el-table-column prop="locale" label="语言" width="100" />
        <el-table-column label="创建时间" width="180"><template #default="{ row }">{{ formatShanghaiDateTime(row.created_at) }}</template></el-table-column>
        <el-table-column label="操作" width="90" fixed="right" align="center"><template #default="{ row }"><el-dropdown trigger="click" @command="(command: string) => handleCommand(row, command)" @click.stop><el-button link :icon="MoreFilled" @click.stop /><template #dropdown><el-dropdown-menu><el-dropdown-item command="detail">查看详情</el-dropdown-item><template v-if="canManage && !row.is_default"><el-dropdown-item v-if="row.status !== 'suspended'" command="suspend" divided>暂停租户</el-dropdown-item><el-dropdown-item v-if="row.status !== 'active'" command="activate">恢复租户</el-dropdown-item><el-dropdown-item v-if="row.status !== 'disabled'" command="disable">停用租户</el-dropdown-item></template></el-dropdown-menu></template></el-dropdown></template></el-table-column>
      </el-table>
      <footer class="table-footer"><span>共 {{ total }} 家企业</span><el-pagination v-model:current-page="query.page" v-model:page-size="query.page_size" :total="total" layout="prev, pager, next, sizes" :page-sizes="[10, 20, 50, 100]" @current-change="load" @size-change="search" /></footer>
    </article>

    <el-dialog v-model="createVisible" title="创建企业租户" width="560px" destroy-on-close>
      <el-alert type="info" title="创建后将生成不可变的 Tenant Key；企业标识用于运营检索和路由识别。" :closable="false" show-icon />
      <el-form class="dialog-form" label-position="top"><el-form-item label="企业名称" required><el-input v-model="form.name" maxlength="128" show-word-limit /></el-form-item><el-form-item label="唯一标识" required><el-input v-model="form.slug" placeholder="例如 acme-china" maxlength="64" /></el-form-item><div class="two-columns"><el-form-item label="时区"><el-input v-model="form.timezone" /></el-form-item><el-form-item label="语言"><el-input v-model="form.locale" /></el-form-item></div></el-form>
      <template #footer><el-button @click="createVisible = false">取消</el-button><el-button type="primary" :loading="creating" @click="submitCreate">创建并查看详情</el-button></template>
    </el-dialog>

    <el-dialog v-model="statusVisible" title="变更租户状态" width="520px">
      <el-alert type="warning" :title="`将“${selectedTenant?.name || ''}”变更为“${statusMeta(targetStatus).label}”`" description="暂停或停用会影响企业成员访问；恢复不会自动恢复已单独暂停的成员。" :closable="false" show-icon />
      <el-form class="dialog-form" label-position="top"><el-form-item label="变更原因" required><el-input v-model="statusReason" type="textarea" :rows="4" maxlength="500" show-word-limit placeholder="说明业务背景、恢复条件或审批依据" /></el-form-item></el-form>
      <template #footer><el-button @click="statusVisible = false">取消</el-button><el-button type="primary" @click="submitStatusChange">确认变更</el-button></template>
    </el-dialog>
  </section>
</template>
