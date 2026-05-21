<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { createInviteCode, extendInviteCode, listInviteCodes, reactivateInviteCode, revokeInviteCode } from '@/api/admin'
import type { InviteCodeInfo } from '@/types/domain'

const list = ref<InviteCodeInfo[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const errorMessage = ref('')
const dialogVisible = ref(false)
const saving = ref(false)
const form = reactive({ expires_at: '' })
const extendingId = ref(0)
const extendingVisible = ref(false)
const extendForm = reactive({ new_expires_at: '' })

const load = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    const data = await listInviteCodes(page.value, pageSize.value)
    list.value = data.list || []
    total.value = data.total || 0
  } catch (e: unknown) {
    errorMessage.value = (e as { message?: string }).message || '加载失败'
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  form.expires_at = ''
  dialogVisible.value = true
}

const saveCreate = async () => {
  saving.value = true
  try {
    await createInviteCode(form.expires_at || undefined)
    ElMessage.success('邀请码已生成')
    dialogVisible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

const openExtend = (row: InviteCodeInfo) => {
  extendingId.value = row.id
  extendForm.new_expires_at = row.expires_at || ''
  extendingVisible.value = true
}

const saveExtend = async () => {
  if (!extendForm.new_expires_at) {
    ElMessage.warning('请选择新的过期时间')
    return
  }
  saving.value = true
  try {
    await extendInviteCode(extendingId.value, new Date(extendForm.new_expires_at).toISOString())
    ElMessage.success('有效期已延长')
    extendingVisible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

const handleToggleActive = async (row: InviteCodeInfo) => {
  if (row.is_active) {
    try {
      await ElMessageBox.confirm(`确认撤销邀请码 ${row.code}？撤销后该邀请码将无法用于注册。`, '撤销邀请码', {
        confirmButtonText: '撤销',
        cancelButtonText: '取消',
        type: 'warning',
      })
    } catch {
      return
    }
    try {
      await revokeInviteCode(row.id)
      ElMessage.success('邀请码已撤销')
      await load()
    } catch (e: unknown) {
      ElMessage.error((e as { message?: string }).message || '撤销失败')
    }
  } else {
    try {
      await reactivateInviteCode(row.id)
      ElMessage.success('邀请码已重启')
      await load()
    } catch (e: unknown) {
      ElMessage.error((e as { message?: string }).message || '重启失败')
    }
  }
}

const statusTag = (row: InviteCodeInfo) => {
  if (!row.is_active) return { text: '已撤销', type: 'danger' as const }
  if (row.expires_at && new Date(row.expires_at) < new Date()) return { text: '已过期', type: 'info' as const }
  return { text: '有效', type: 'success' as const }
}

const formatTime = (s?: string) => {
  if (!s) return '永不过期'
  return new Date(s).toLocaleString('zh-CN')
}

const copyLink = async (row: InviteCodeInfo) => {
  const url = `${window.location.origin}/register?invite_code=${row.code}`
  try {
    await navigator.clipboard.writeText(url)
    ElMessage.success('注册链接已复制到剪贴板')
  } catch {
    ElMessage.warning('复制失败，请手动复制：' + url)
  }
}

const isExpired = (row: InviteCodeInfo) =>
  !row.is_active || (!!row.expires_at && new Date(row.expires_at) < new Date())

const onPageChange = (p: number) => { page.value = p; load() }
const onSizeChange = (s: number) => { pageSize.value = s; page.value = 1; load() }

onMounted(load)
</script>

<template>
  <section class="invite-code-page">
    <div class="page-header">
      <h1 class="page-title">邀请码管理</h1>
      <div class="toolbar">
        <el-button type="primary" @click="openCreate">生成邀请码</el-button>
      </div>
    </div>
    <div class="content-surface invite-code-surface">
      <el-alert v-if="errorMessage" class="page-error" type="error" :title="errorMessage" show-icon :closable="false">
        <template #default>
          <el-button type="primary" size="small" @click="load">重试</el-button>
        </template>
      </el-alert>
      <div class="invite-code-table-area desktop-only">
        <el-table v-loading="loading" :data="list" empty-text="暂无邀请码" height="100%">
          <el-table-column prop="code" label="邀请码" min-width="280" />
          <el-table-column label="状态" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="statusTag(row).type" size="small">{{ statusTag(row).text }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="过期时间" width="180" align="center">
            <template #default="{ row }">{{ formatTime(row.expires_at) }}</template>
          </el-table-column>
          <el-table-column label="创建时间" width="180" align="center">
            <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="320" fixed="right" align="center">
            <template #default="{ row }">
              <el-button text type="primary" size="small" :disabled="isExpired(row)" @click="copyLink(row)">复制链接</el-button>
              <el-button text type="primary" size="small" :disabled="!row.is_active" @click="openExtend(row)">延长</el-button>
              <el-button text :type="row.is_active ? 'danger' : 'success'" size="small" @click="handleToggleActive(row)">{{ row.is_active ? '撤销' : '重启' }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <!-- Mobile invite code cards -->
      <div class="mobile-card-list mobile-only">
        <el-empty v-if="!loading && list.length === 0" description="暂无邀请码" />
        <div v-for="row in list" :key="row.id" class="mobile-invite-card">
          <div class="mobile-card__header">
            <div><span class="invite-code__label">邀请码：</span><span class="invite-code__text">{{ row.code }}</span></div>
            <el-tag :type="statusTag(row).type" size="small">{{ statusTag(row).text }}</el-tag>
          </div>
          <div class="mobile-card__meta">
            <span>过期：{{ formatTime(row.expires_at) }}</span>
          </div>
          <div class="mobile-card__meta">
            <span>创建：{{ formatTime(row.created_at) }}</span>
          </div>
          <div class="mobile-card__actions mobile-invite-card__actions">
            <el-button size="small" type="primary" plain :disabled="isExpired(row)" @click="copyLink(row)">复制</el-button>
            <el-button size="small" type="primary" plain :disabled="!row.is_active" @click="openExtend(row)">延长</el-button>
            <el-button size="small" :type="row.is_active ? 'danger' : 'success'" plain @click="handleToggleActive(row)">{{ row.is_active ? '撤销' : '重启' }}</el-button>
          </div>
        </div>
      </div>
      <el-pagination
        v-if="total > 0"
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next, sizes"
        style="margin-top: 12px; justify-content: flex-end"
        @current-change="onPageChange"
        @size-change="onSizeChange"
      />
    </div>

    <!-- Create dialog -->
    <el-dialog v-model="dialogVisible" title="生成邀请码" width="460px" @closed="form.expires_at = ''">
      <el-form label-position="top">
        <el-form-item label="过期时间">
          <el-date-picker
            v-model="form.expires_at"
            type="datetime"
            placeholder="留空则永不过期"
            value-format="YYYY-MM-DDTHH:mm:ss"
            style="width: 100%"
          />
        </el-form-item>
        <p style="color: #909399; font-size: 13px; margin-top: -8px">留空则生成永久有效的邀请码。</p>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveCreate">生成</el-button>
      </template>
    </el-dialog>

    <!-- Extend dialog -->
    <el-dialog v-model="extendingVisible" title="延长有效期" width="460px">
      <el-form label-position="top">
        <el-form-item label="新过期时间">
          <el-date-picker
            v-model="extendForm.new_expires_at"
            type="datetime"
            placeholder="选择新的过期时间"
            value-format="YYYY-MM-DDTHH:mm:ss"
            style="width: 100%"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="extendingVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveExtend">确认</el-button>
      </template>
    </el-dialog>
  </section>
</template>
