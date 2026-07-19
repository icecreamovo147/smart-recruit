<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Plus, Search, User } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { createTenant, listMemberships, listTenants, updateTenantStatus } from '@/api/tenant'
import type { Membership, Tenant } from '@/types'

const loading = ref(false)
const creating = ref(false)
const createVisible = ref(false)
const memberVisible = ref(false)
const tenants = ref<Tenant[]>([])
const memberships = ref<Membership[]>([])
const total = ref(0)
const selectedTenant = ref<Tenant | null>(null)
const query = reactive({ page: 1, page_size: 20, keyword: '', status: '' })
const form = reactive({ slug: '', name: '', timezone: 'Asia/Shanghai', locale: 'zh-CN' })

const load = async () => {
  loading.value = true
  try { const response = await listTenants(query); tenants.value = response.list || []; total.value = response.total } finally { loading.value = false }
}
const submitCreate = async () => {
  if (!form.slug.trim() || !form.name.trim()) { ElMessage.warning('请填写企业标识与名称'); return }
  creating.value = true
  try {
    await createTenant({ ...form, slug: form.slug.trim(), name: form.name.trim() })
    createVisible.value = false
    Object.assign(form, { slug: '', name: '', timezone: 'Asia/Shanghai', locale: 'zh-CN' })
    ElMessage.success('企业租户已创建')
    await load()
  } finally { creating.value = false }
}
const changeStatus = async (tenant: Tenant, status: string) => {
  await ElMessageBox.confirm(`确认将“${tenant.name}”状态调整为 ${status}？已签发会话会在权限校验时失效。`, '租户状态变更', { type: 'warning' })
  await updateTenantStatus(tenant.id, status)
  ElMessage.success('状态已更新')
  await load()
}
const openMembers = async (tenant: Tenant) => {
  selectedTenant.value = tenant
  const response = await listMemberships(tenant.id)
  memberships.value = response.list || []
  memberVisible.value = true
}
const statusType = (status: string) => status === 'active' ? 'success' : status === 'suspended' ? 'warning' : 'info'
onMounted(load)
</script>

<template>
  <section class="page">
    <div class="page-header"><div><h1>企业租户</h1><p>管理企业生命周期、应用准入与成员归属。平台身份与企业身份在此保持隔离。</p></div><el-button type="primary" :icon="Plus" @click="createVisible = true">创建企业</el-button></div>
    <el-card shadow="never">
      <div class="filters">
        <el-input v-model="query.keyword" :prefix-icon="Search" clearable placeholder="搜索企业名称或标识" @keyup.enter="query.page = 1; load()" />
        <el-select v-model="query.status" clearable placeholder="全部状态" @change="query.page = 1; load()"><el-option label="正常" value="active" /><el-option label="已暂停" value="suspended" /><el-option label="已停用" value="disabled" /></el-select>
        <el-button @click="query.page = 1; load()">查询</el-button>
      </div>
      <el-table v-loading="loading" :data="tenants">
        <el-table-column prop="name" label="企业"><template #default="{ row }"><strong>{{ row.name }}</strong><small class="secondary">{{ row.slug }} · {{ row.tenant_key }}</small></template></el-table-column>
        <el-table-column label="状态" width="110"><template #default="{ row }"><el-tag :type="statusType(row.status)">{{ row.status }}</el-tag></template></el-table-column>
        <el-table-column prop="membership_count" label="成员数" width="100" />
        <el-table-column prop="timezone" label="时区" width="150" />
        <el-table-column label="创建时间" width="180"><template #default="{ row }">{{ new Date(row.created_at).toLocaleString('zh-CN', { hour12: false }) }}</template></el-table-column>
        <el-table-column label="操作" width="260" fixed="right"><template #default="{ row }"><el-button link type="primary" :icon="User" @click="openMembers(row)">成员</el-button><el-button v-if="row.status !== 'suspended' && !row.is_default" link type="warning" @click="changeStatus(row, 'suspended')">暂停</el-button><el-button v-if="row.status === 'suspended'" link type="success" @click="changeStatus(row, 'active')">恢复</el-button><el-tag v-if="row.is_default" type="info">默认租户</el-tag></template></el-table-column>
      </el-table>
      <el-pagination v-model:current-page="query.page" :page-size="query.page_size" :total="total" layout="total, prev, pager, next" @current-change="load" />
    </el-card>

    <el-dialog v-model="createVisible" title="创建企业租户" width="520px"><el-form label-position="top"><el-form-item label="企业名称" required><el-input v-model="form.name" /></el-form-item><el-form-item label="唯一标识" required><el-input v-model="form.slug" placeholder="例如 acme-china" /></el-form-item><div class="two-columns"><el-form-item label="时区"><el-input v-model="form.timezone" /></el-form-item><el-form-item label="语言"><el-input v-model="form.locale" /></el-form-item></div></el-form><template #footer><el-button @click="createVisible = false">取消</el-button><el-button type="primary" :loading="creating" @click="submitCreate">创建</el-button></template></el-dialog>
    <el-dialog v-model="memberVisible" :title="`${selectedTenant?.name || ''} · 成员`" width="760px"><el-table :data="memberships" empty-text="暂无成员"><el-table-column prop="username" label="账号" /><el-table-column label="角色"><template #default="{ row }">{{ row.roles.join('、') || '-' }}</template></el-table-column><el-table-column prop="membership_status" label="成员状态" width="120" /><el-table-column label="数据范围"><template #default="{ row }">{{ row.data_scopes.join('、') || '默认范围' }}</template></el-table-column></el-table></el-dialog>
  </section>
</template>
