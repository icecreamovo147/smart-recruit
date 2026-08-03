<script setup lang="ts">
import { t } from '@shared/i18n'
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown, MoreFilled, Plus, Refresh, Search } from '@element-plus/icons-vue'
import { createInviteCode, extendInviteCode, listInviteCodes, reactivateInviteCode, revokeInviteCode } from '@/api/admin'
import type { InviteCodeInfo } from '@/types/domain'
import { formatShanghaiDateTime, toShanghaiRFC3339 } from '@shared/utils/format'

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
const keyword = ref('')
const statusFilter = ref('')

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
    ElMessage.success(t('common.success'))
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
    ElMessage.warning(t('common.invalid_request'))
    return
  }
  saving.value = true
  try {
    await extendInviteCode(extendingId.value, toShanghaiRFC3339(extendForm.new_expires_at))
    ElMessage.success(t('common.success'))
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
      ElMessage.success(t('common.success'))
      await load()
    } catch (e: unknown) {
      ElMessage.error((e as { message?: string }).message || '撤销失败')
    }
  } else {
    try {
      await reactivateInviteCode(row.id)
      ElMessage.success(t('common.success'))
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
	return formatShanghaiDateTime(s)
}

const copyLink = async (row: InviteCodeInfo) => {
  const url = `${window.location.origin}/register?invite_code=${row.code}`
  try {
    await navigator.clipboard.writeText(url)
    ElMessage.success(t('common.success'))
  } catch {
    ElMessage.warning(t('common.invalid_request'))
  }
}

const isExpired = (row: InviteCodeInfo) =>
  !row.is_active || (!!row.expires_at && new Date(row.expires_at) < new Date())

const filteredList = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return list.value.filter((item) => {
    const tag = statusTag(item).text
    const matchesKeyword = !q || item.code.toLowerCase().includes(q)
    const matchesStatus = !statusFilter.value || tag === statusFilter.value
    return matchesKeyword && matchesStatus
  })
})

const onPageChange = (p: number) => { page.value = p; load() }
const onSizeChange = (s: number) => { pageSize.value = s; page.value = 1; load() }

onMounted(load)
</script>

<template>
  <section class="console-page console-page--fill invite-code-page">
    <div class="workspace-surface">
      <div class="workspace-surface__header">
        <div class="workspace-surface__header-copy">
          <p class="console-eyebrow">ACCESS CONTROL</p>
          <h1 class="console-title">邀请码管理</h1>
          <p class="console-description">生成和管理 HR 注册入口的邀请码，控制账号创建边界，并跟踪有效、过期和撤销状态。</p>
        </div>
        <div class="workspace-surface__header-actions">
          <el-button :icon="Refresh" @click="load">刷新</el-button>
          <el-button type="primary" :icon="Plus" @click="openCreate">生成邀请码</el-button>
        </div>
      </div>

      <div class="workspace-surface__divider"></div>

      <div class="workspace-surface__toolbar">
        <div class="workspace-surface__filters">
          <el-input v-model="keyword" :prefix-icon="Search" clearable placeholder="搜索邀请码" style="width: 240px" />
          <el-select v-model="statusFilter" clearable placeholder="全部状态" style="width: 140px">
            <el-option label="有效" value="有效" />
            <el-option label="已过期" value="已过期" />
            <el-option label="已撤销" value="已撤销" />
          </el-select>
        </div>
      </div>

      <el-alert v-if="errorMessage" class="workspace-surface__error" type="error" :title="errorMessage" show-icon :closable="false">
        <template #default>
          <el-button type="primary" size="small" @click="load">重试</el-button>
        </template>
      </el-alert>

      <div class="workspace-surface__body desktop-only">
        <el-table v-loading="loading" :data="filteredList" class="console-table" empty-text="暂无邀请码" height="100%">
          <el-table-column label="邀请码" min-width="280">
            <template #default="{ row }">
              <div class="console-entity">
                <div class="console-entity__name console-code">{{ row.code }}</div>
                <div class="console-entity__meta">注册链接可复制给 HR 用户完成注册</div>
              </div>
            </template>
          </el-table-column>
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
          <el-table-column label="操作" width="200" fixed="right" align="center">
            <template #default="{ row }">
              <el-button size="small" :disabled="isExpired(row)" @click="copyLink(row)">复制链接</el-button>
              <el-dropdown trigger="click" @command="(cmd: string) => { if (cmd === 'extend') openExtend(row); if (cmd === 'toggle') handleToggleActive(row) }">
                <el-button size="small">更多<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="extend" :disabled="!row.is_active">延长</el-dropdown-item>
                    <el-dropdown-item v-if="row.is_active" command="toggle" divided style="color: var(--el-color-danger)">撤销</el-dropdown-item>
                    <el-dropdown-item v-else command="toggle" divided>重启</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div class="workspace-surface__pagination">
        <el-pagination
          v-if="total > 0"
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          layout="total, prev, pager, next, sizes"
          @current-change="onPageChange"
          @size-change="onSizeChange"
        />
      </div>

      <div class="mobile-card-list mobile-only">
        <el-empty v-if="!loading && filteredList.length === 0" description="暂无邀请码" />
        <div v-for="row in filteredList" :key="row.id" class="mobile-invite-card">
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
    </div>

    <!-- Create dialog -->
    <el-drawer v-model="dialogVisible" title="生成邀请码" size="460px" :close-on-click-modal="true" @closed="form.expires_at = ''">
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
        <p style="color: var(--el-text-color-secondary); font-size: 13px; margin-top: -8px">留空则生成永久有效的邀请码。</p>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveCreate">生成</el-button>
      </template>
    </el-drawer>

    <!-- Extend dialog -->
    <el-drawer v-model="extendingVisible" title="延长有效期" size="460px" :close-on-click-modal="true">
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
    </el-drawer>
  </section>
</template>
