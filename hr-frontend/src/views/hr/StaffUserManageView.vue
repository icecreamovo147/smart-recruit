<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  assignDataScope,
  assignUserRole,
  createStaffUser,
  getUserRoles,
  listPermissions,
  listRoles,
  listStaffUsers,
  revokeDataScope,
  revokeUserRole,
} from '@/api/admin'
import { PERM } from '@/types/domain'
import type {
  DataScopeInfo,
  PermissionInfo,
  RoleInfo,
  StaffUserInfo,
} from '@/types/domain'

// ── Staff user list ──────────────────────────────────────────────────────

const list = ref<StaffUserInfo[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const errorMessage = ref('')

const load = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    const data = await listStaffUsers({ page: page.value, page_size: pageSize.value })
    list.value = data.list || []
    total.value = data.total || 0
  } catch (e: unknown) {
    errorMessage.value = (e as { message?: string }).message || '加载失败'
  } finally {
    loading.value = false
  }
}

const onPageChange = (p: number) => { page.value = p; load() }
const onSizeChange = (s: number) => { pageSize.value = s; page.value = 1; load() }

onMounted(load)

// ── Create user dialog ───────────────────────────────────────────────────

const createDialogVisible = ref(false)
const saving = ref(false)
const createForm = reactive({
  username: '',
  password: '',
  email: '',
  role_keys: [] as string[],
})
const allRoles = ref<RoleInfo[]>([])

const openCreateDialog = async () => {
  createForm.username = ''
  createForm.password = ''
  createForm.email = ''
  createForm.role_keys = []
  if (allRoles.value.length === 0) {
    try {
      const data = await listRoles()
      allRoles.value = data.list || []
    } catch { /* ignore */ }
  }
  createDialogVisible.value = true
}

const saveCreate = async () => {
  if (!createForm.username) {
    ElMessage.warning('请输入用户名')
    return
  }
  if (!createForm.password) {
    ElMessage.warning('请输入密码')
    return
  }
  saving.value = true
  try {
    await createStaffUser({
      username: createForm.username,
      password: createForm.password,
      email: createForm.email || undefined,
      role_keys: createForm.role_keys.length > 0 ? createForm.role_keys : undefined,
    })
    ElMessage.success('员工账号已创建')
    createDialogVisible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

// ── Role management dialog ───────────────────────────────────────────────

const roleDialogVisible = ref(false)
const targetUser = ref<StaffUserInfo | null>(null)
const userRoleKeys = ref<string[]>([])
const userPermKeys = ref<string[]>([])
const roleSaving = ref(false)

const openRoleDialog = async (user: StaffUserInfo) => {
  targetUser.value = user
  userRoleKeys.value = []
  userPermKeys.value = []
  if (allRoles.value.length === 0) {
    try {
      const data = await listRoles()
      allRoles.value = data.list || []
    } catch { /* ignore */ }
  }
  try {
    const data = await getUserRoles(user.user_id)
    userRoleKeys.value = data.role_keys || []
    userPermKeys.value = data.permission_keys || []
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '获取用户角色失败')
  }
  roleDialogVisible.value = true
}

const toggleRole = async (roleKey: string) => {
  if (!targetUser.value) return
  roleSaving.value = true
  try {
    if (userRoleKeys.value.includes(roleKey)) {
      await revokeUserRole(targetUser.value.user_id, roleKey)
      userRoleKeys.value = userRoleKeys.value.filter((k) => k !== roleKey)
      ElMessage.success('角色已移除')
    } else {
      await assignUserRole(targetUser.value.user_id, roleKey)
      userRoleKeys.value.push(roleKey)
      ElMessage.success('角色已分配')
    }
    // Refresh permissions after role change
    try {
      const data = await getUserRoles(targetUser.value.user_id)
      userPermKeys.value = data.permission_keys || []
    } catch { /* ignore */ }
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '操作失败')
  } finally {
    roleSaving.value = false
  }
}

// ── Data scope dialog ────────────────────────────────────────────────────

const scopeDialogVisible = ref(false)
const userScopes = ref<DataScopeInfo[]>([])
const scopeSaving = ref(false)
const allScopeKeys = ref<string[]>(['own_jobs', 'department', 'location', 'recruiting_all', 'assigned_interviews'])
const scopeDisplayMap: Record<string, string> = {
  own_jobs: '自己负责的岗位',
  department: '指定部门',
  location: '指定地点',
  recruiting_all: '全部招聘数据',
  assigned_interviews: '被分配的面试',
}
const newScopeForm = reactive({
  scope_key: '',
  resource_type: '',
  resource_id: undefined as number | undefined,
})

// Auto-assign resource_type when scope_key is 'department' or 'location'.
// This ensures the backend can look up the correct resource_id for the scope.
watch(() => newScopeForm.scope_key, (key) => {
  if (key === 'department') {
    newScopeForm.resource_type = 'department'
  } else if (key === 'location') {
    newScopeForm.resource_type = 'location'
  } else {
    newScopeForm.resource_type = ''
  }
})

const openScopeDialog = async (user: StaffUserInfo) => {
  targetUser.value = user
  userScopes.value = []
  try {
    const data = await getUserRoles(user.user_id)
    userScopes.value = data.data_scopes || []
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '获取数据范围失败')
  }
  scopeDialogVisible.value = true
}

const addScope = async () => {
  if (!targetUser.value) return
  if (!newScopeForm.scope_key) {
    ElMessage.warning('请选择数据范围')
    return
  }
  scopeSaving.value = true
  try {
    await assignDataScope(
      targetUser.value.user_id,
      newScopeForm.scope_key,
      newScopeForm.resource_type || undefined,
      newScopeForm.resource_id || undefined,
    )
    ElMessage.success('数据范围已分配')
    // Refresh
    const data = await getUserRoles(targetUser.value.user_id)
    userScopes.value = data.data_scopes || []
    newScopeForm.scope_key = ''
    newScopeForm.resource_type = ''
    newScopeForm.resource_id = undefined
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '操作失败')
  } finally {
    scopeSaving.value = false
  }
}

const removeScope = async (scope: DataScopeInfo) => {
  try {
    await ElMessageBox.confirm('确认移除该数据范围？', '移除数据范围', {
      confirmButtonText: '移除',
      cancelButtonText: '取消',
      type: 'warning',
    })
  } catch {
    return
  }
  scopeSaving.value = true
  try {
    await revokeDataScope(scope.id)
    userScopes.value = userScopes.value.filter((s) => s.id !== scope.id)
    ElMessage.success('数据范围已移除')
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '操作失败')
  } finally {
    scopeSaving.value = false
  }
}

// ── Helpers ──────────────────────────────────────────────────────────────

const formatTime = (s?: string) => {
  if (!s) return '-'
  return new Date(s).toLocaleString('zh-CN')
}

const statusTag = (user: StaffUserInfo) => {
  if (user.status === 'disabled' || user.status === 'locked') return { text: user.status === 'disabled' ? '已禁用' : '已锁定', type: 'danger' as const }
  if (user.status === 'pending') return { text: '待激活', type: 'warning' as const }
  return { text: '正常', type: 'success' as const }
}
</script>

<template>
  <section class="staff-user-page">
    <div class="page-header">
      <h1 class="page-title">员工账号管理</h1>
      <div class="toolbar">
        <el-button type="primary" @click="openCreateDialog">创建员工账号</el-button>
      </div>
    </div>
    <div class="content-surface">
      <el-alert v-if="errorMessage" class="page-error" type="error" :title="errorMessage" show-icon :closable="false">
        <template #default>
          <el-button type="primary" size="small" @click="load">重试</el-button>
        </template>
      </el-alert>
      <el-table v-loading="loading" :data="list" empty-text="暂无员工账号" height="100%">
        <el-table-column prop="user_id" label="ID" width="80" align="center" />
        <el-table-column prop="username" label="用户名" min-width="140" />
        <el-table-column prop="email" label="邮箱" min-width="180">
          <template #default="{ row }">{{ row.email || '-' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTag(row).type" size="small">{{ statusTag(row).text }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="角色" min-width="200">
          <template #default="{ row }">
            <el-tag
              v-for="rk in (row.roles || [])"
              :key="rk"
              size="small"
              style="margin-right: 4px; margin-bottom: 2px"
            >{{ rk }}</el-tag>
            <span v-if="!row.roles || row.roles.length === 0" style="color: #909399">无</span>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="180" align="center">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="240" fixed="right" align="center">
          <template #default="{ row }">
            <el-button text type="primary" size="small" @click="openRoleDialog(row)">角色</el-button>
            <el-button text type="primary" size="small" @click="openScopeDialog(row)">数据范围</el-button>
          </template>
        </el-table-column>
      </el-table>
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

    <!-- Create user dialog -->
    <el-dialog v-model="createDialogVisible" title="创建员工账号" width="480px">
      <el-form label-position="top">
        <el-form-item label="用户名" required>
          <el-input v-model="createForm.username" placeholder="输入登录用户名" />
        </el-form-item>
        <el-form-item label="密码" required>
          <el-input v-model="createForm.password" type="password" placeholder="输入密码" show-password />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="createForm.email" placeholder="选填" />
        </el-form-item>
        <el-form-item label="初始角色">
          <el-checkbox-group v-model="createForm.role_keys">
            <el-checkbox v-for="role in allRoles" :key="role.role_key" :label="role.role_key" :value="role.role_key">
              {{ role.name }}
            </el-checkbox>
          </el-checkbox-group>
          <div v-if="allRoles.length === 0" style="color: #909399; font-size: 13px; margin-top: 4px">
            加载角色列表失败，创建后可在角色管理中分配
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveCreate">创建</el-button>
      </template>
    </el-dialog>

    <!-- Role management dialog -->
    <el-dialog v-model="roleDialogVisible" title="角色管理" width="560px">
      <template v-if="targetUser">
        <p style="margin-bottom: 12px; color: #606266">
          用户：<strong>{{ targetUser.username }}</strong>（ID: {{ targetUser.user_id }}）
        </p>
        <el-table :data="allRoles" style="margin-bottom: 16px" max-height="320">
          <el-table-column prop="name" label="角色" min-width="120" />
          <el-table-column prop="description" label="描述" min-width="180" />
          <el-table-column label="当前状态" width="100" align="center">
            <template #default="{ row }">
              <el-tag v-if="userRoleKeys.includes(row.role_key)" type="success" size="small">已分配</el-tag>
              <el-tag v-else type="info" size="small">未分配</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="110" align="center">
            <template #default="{ row }">
              <el-button
                v-if="userRoleKeys.includes(row.role_key)"
                text
                type="danger"
                size="small"
                :loading="roleSaving"
                @click="toggleRole(row.role_key)"
              >移除</el-button>
              <el-button
                v-else
                text
                type="primary"
                size="small"
                :loading="roleSaving"
                @click="toggleRole(row.role_key)"
              >分配</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-collapse>
          <el-collapse-item title="当前权限列表" name="permissions">
            <div v-if="userPermKeys.length > 0" style="display: flex; flex-wrap: wrap; gap: 4px">
              <el-tag v-for="pk in userPermKeys" :key="pk" size="small" type="info">{{ pk }}</el-tag>
            </div>
            <span v-else style="color: #909399">暂无权限</span>
          </el-collapse-item>
        </el-collapse>
      </template>
      <template #footer>
        <el-button @click="roleDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- Data scope dialog -->
    <el-dialog v-model="scopeDialogVisible" title="数据范围管理" width="560px">
      <template v-if="targetUser">
        <p style="margin-bottom: 12px; color: #606266">
          用户：<strong>{{ targetUser.username }}</strong>（ID: {{ targetUser.user_id }}）
        </p>
        <h4 style="margin-bottom: 8px">当前已分配的数据范围</h4>
        <el-table :data="userScopes" style="margin-bottom: 16px" empty-text="暂无数据范围">
          <el-table-column prop="scope_key" label="范围" min-width="140">
            <template #default="{ row }">
              {{ scopeDisplayMap[row.scope_key] || row.scope_key }}
            </template>
          </el-table-column>
          <el-table-column prop="resource_type" label="资源类型" width="110" />
          <el-table-column prop="resource_id" label="资源ID" width="80" align="center">
            <template #default="{ row }">{{ row.resource_id || '-' }}</template>
          </el-table-column>
          <el-table-column label="操作" width="80" align="center">
            <template #default="{ row }">
              <el-button text type="danger" size="small" :loading="scopeSaving" @click="removeScope(row)">移除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-divider />
        <h4 style="margin-bottom: 8px">添加新数据范围</h4>
        <el-form label-position="top">
          <el-form-item label="范围类型" required>
            <el-select v-model="newScopeForm.scope_key" placeholder="选择数据范围" style="width: 100%">
              <el-option
                v-for="sk in allScopeKeys"
                :key="sk"
                :label="scopeDisplayMap[sk] || sk"
                :value="sk"
              />
            </el-select>
          </el-form-item>
          <el-form-item v-if="newScopeForm.scope_key === 'department'" label="资源类型">
            <el-input v-model="newScopeForm.resource_type" placeholder="固定值: department" disabled />
          </el-form-item>
          <el-form-item v-if="newScopeForm.scope_key === 'location'" label="资源类型">
            <el-input v-model="newScopeForm.resource_type" placeholder="固定值: location" disabled />
          </el-form-item>
          <el-form-item v-if="newScopeForm.scope_key === 'department' || newScopeForm.scope_key === 'location'" label="资源ID">
            <el-input-number v-model="newScopeForm.resource_id" :min="1" placeholder="输入资源ID" style="width: 100%" />
          </el-form-item>
        </el-form>
        <el-button type="primary" :loading="scopeSaving" @click="addScope">添加</el-button>
      </template>
      <template #footer>
        <el-button @click="scopeDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </section>
</template>
