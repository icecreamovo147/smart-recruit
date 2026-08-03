<script setup lang="ts">
import { t } from '@shared/i18n'
import { onMounted, reactive, ref } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { createPlatformUser, listPlatformUsers, updatePlatformUser } from '@/api/platform-user'
import { DataTableCard, FilterToolbar, PageHeader, PagePanel } from '@/components/admin-console'
import { roleLabel } from '@/permissions'
import type { PlatformAccount } from '@/types'
import { formatShanghaiDateTime } from '@shared/utils/format'

const loading = ref(false)
const rows = ref<PlatformAccount[]>([])
const total = ref(0)
const query = reactive({ status: '', page: 1, page_size: 20 })
const createVisible = ref(false)
const editVisible = ref(false)
const selected = ref<PlatformAccount | null>(null)
const createForm = reactive({ username: '', email: '', password: '', role_key: 'platform_operator' })
const editForm = reactive({ role_key: 'platform_operator', status: 'active', reason: '' })
const roles = ['platform_admin', 'platform_operator', 'platform_auditor']

const load = async () => { loading.value = true; try { const response = await listPlatformUsers(query); rows.value = response.list || []; total.value = Number(response.total) || 0 } finally { loading.value = false } }
const openCreate = () => { Object.assign(createForm, { username: '', email: '', password: '', role_key: 'platform_operator' }); createVisible.value = true }
const submitCreate = async () => {
  if (!createForm.username.trim() || createForm.password.length < 8) { ElMessage.warning(t('common.invalid_request')); return }
  await createPlatformUser(createForm); createVisible.value = false; ElMessage.success(t('common.success')); await load()
}
const openEdit = (row: PlatformAccount) => { selected.value = row; editForm.role_key = row.roles[0] || 'platform_operator'; editForm.status = row.status; editForm.reason = ''; editVisible.value = true }
const submitEdit = async () => {
  if (!selected.value || !editForm.reason.trim()) { ElMessage.warning(t('common.invalid_request')); return }
  await updatePlatformUser(selected.value.user_id, { ...editForm, reason: editForm.reason.trim() }); editVisible.value = false; ElMessage.success(t('common.success')); await load()
}
const formatTime = (value?: string) => formatShanghaiDateTime(value)
onMounted(load)
</script>

<template>
  <section class="console-page" v-loading="loading">
    <PagePanel>
      <PageHeader title="平台账号" kicker="PLATFORM ACCOUNTS" description="管理平台运营账号、角色与状态" />

      <FilterToolbar>
        <div class="filter-fields">
          <el-select v-model="query.status" clearable placeholder="全部状态" @change="load"><el-option label="有效" value="active" /><el-option label="已停用" value="disabled" /></el-select>
        </div>
        <template #actions>
          <div class="filter-actions"><el-button type="primary" :icon="Plus" @click="openCreate">新建账号</el-button></div>
        </template>
      </FilterToolbar>

      <DataTableCard :result-count="total" :result-label="`共 ${total} 个平台账号`">
        <el-table :data="rows" stripe class="console-table">
          <el-table-column label="账号" min-width="220"><template #default="{ row }"><div class="user-cell"><span>{{ row.username.slice(0,1).toUpperCase() }}</span><div><strong>{{ row.username }}</strong><small>{{ row.email || `用户 ID ${row.user_id}` }}</small></div></div></template></el-table-column>
          <el-table-column label="平台角色" min-width="180"><template #default="{ row }"><el-tag v-for="role in row.roles" :key="role" type="info">{{ roleLabel(role) }}</el-tag></template></el-table-column>
          <el-table-column label="状态" width="110"><template #default="{ row }"><el-tag :type="row.status === 'active' ? 'success' : 'info'">{{ row.status === 'active' ? '有效' : '已停用' }}</el-tag></template></el-table-column>
          <el-table-column label="令牌版本" width="110" prop="token_version" /><el-table-column label="创建时间" width="180"><template #default="{ row }">{{ formatTime(row.created_at) }}</template></el-table-column><el-table-column label="操作" width="100" align="center"><template #default="{ row }"><el-button link type="primary" @click="openEdit(row)">编辑</el-button></template></el-table-column>
        </el-table>
        <template #footer>
          <el-pagination v-model:current-page="query.page" v-model:page-size="query.page_size" :total="total" layout="prev, pager, next, sizes" :page-sizes="[10,20,50]" @current-change="load" @size-change="load" />
        </template>
      </DataTableCard>
    </PagePanel>

    <el-dialog v-model="createVisible" title="新建平台账号" width="560px"><el-form label-position="top"><div class="two-columns"><el-form-item label="用户名" required><el-input v-model="createForm.username" /></el-form-item><el-form-item label="邮箱"><el-input v-model="createForm.email" /></el-form-item></div><el-form-item label="初始密码" required><el-input v-model="createForm.password" type="password" show-password /></el-form-item><el-form-item label="平台角色" required><el-select v-model="createForm.role_key" style="width:100%"><el-option v-for="role in roles" :key="role" :label="roleLabel(role)" :value="role" /></el-select></el-form-item></el-form><template #footer><el-button @click="createVisible = false">取消</el-button><el-button type="primary" @click="submitCreate">创建</el-button></template></el-dialog>
    <el-dialog v-model="editVisible" title="编辑平台账号" width="560px"><el-alert title="变更角色或状态后，该账号的已签发令牌将失效。" type="warning" :closable="false" show-icon /><el-form class="dialog-form" label-position="top"><div class="two-columns"><el-form-item label="平台角色" required><el-select v-model="editForm.role_key" style="width:100%"><el-option v-for="role in roles" :key="role" :label="roleLabel(role)" :value="role" /></el-select></el-form-item><el-form-item label="账号状态" required><el-select v-model="editForm.status" style="width:100%"><el-option label="有效" value="active" /><el-option label="已停用" value="disabled" /></el-select></el-form-item></div><el-form-item label="变更原因" required><el-input v-model="editForm.reason" type="textarea" :rows="4" maxlength="500" show-word-limit /></el-form-item></el-form><template #footer><el-button @click="editVisible = false">取消</el-button><el-button type="primary" @click="submitEdit">确认变更</el-button></template></el-dialog>
  </section>
</template>
