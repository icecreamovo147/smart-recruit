<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getAuthAuditLogs } from '@/api/analytics'
import type { AuthAuditLogItem } from '@/types/analytics'

const loading = ref(false)
const errorMessage = ref('')
const logs = ref<AuthAuditLogItem[]>([])
const total = ref(0)

const query = reactive({
  page: 1,
  page_size: 20,
  actor_user_id: undefined as number | undefined,
  permission_key: '',
  decision: '',
})

const handleSearch = () => {
  query.page = 1
  load()
}

const handleReset = () => {
  query.actor_user_id = undefined
  query.permission_key = ''
  query.decision = ''
  query.page = 1
  load()
}

const load = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    const data = await getAuthAuditLogs({
      page: query.page,
      page_size: query.page_size,
      actor_user_id: query.actor_user_id,
      permission_key: query.permission_key || undefined,
      decision: query.decision || undefined,
    })
    logs.value = data.list || []
    total.value = Number(data.total) || 0
  } catch (error: unknown) {
    errorMessage.value = error instanceof Error ? error.message : '加载安全审计日志失败'
  } finally {
    loading.value = false
  }
}

const copyText = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

const DECISION_TAG_TYPE: Record<string, string> = {
  allowed: 'success',
  denied: 'danger',
}

onMounted(load)
</script>

<template>
  <div class="security-audit">
    <el-form :inline="true" class="filter-form" @submit.prevent="handleSearch">
      <el-form-item label="操作用户ID">
        <el-input
          v-model.number="query.actor_user_id"
          placeholder="用户ID"
          style="width: 120px"
          clearable
        />
      </el-form-item>
      <el-form-item label="权限">
        <el-input
          v-model="query.permission_key"
          placeholder="权限key"
          style="width: 180px"
          clearable
        />
      </el-form-item>
      <el-form-item label="决策">
        <el-select v-model="query.decision" clearable placeholder="全部" style="width: 120px">
          <el-option label="允许" value="allowed" />
          <el-option label="拒绝" value="denied" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
      </el-form-item>
    </el-form>

    <el-alert
      v-if="errorMessage"
      class="page-error"
      type="error"
      :title="errorMessage"
      show-icon
      :closable="false"
    >
      <template #default>
        <el-button size="small" type="danger" plain @click="load">重试</el-button>
      </template>
    </el-alert>

    <div class="table-wrapper">
      <el-table
        v-loading="loading"
        :data="logs"
        empty-text="暂无安全审计日志"
        stripe
        height="100%"
      >
        <el-table-column prop="created_at" label="时间" min-width="170" align="center">
          <template #default="{ row }">
            {{ row.created_at?.replace('T', ' ').slice(0, 19) || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="actor_user_id" label="操作用户ID" width="110" align="center" />
        <el-table-column prop="actor_roles" label="角色" min-width="150" show-overflow-tooltip />
        <el-table-column prop="permission_key" label="权限" min-width="180" show-overflow-tooltip />
        <el-table-column prop="resource_type" label="资源类型" width="100" align="center" />
        <el-table-column prop="resource_id" label="资源ID" width="80" align="center" />
        <el-table-column prop="decision" label="决策" width="80" align="center">
          <template #default="{ row }">
            <el-tag
              :type="(DECISION_TAG_TYPE[row.decision] as any) || 'info'"
              size="small"
            >
              {{ row.decision === 'allowed' ? '允许' : '拒绝' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="reason" label="原因" min-width="200" show-overflow-tooltip />
        <el-table-column prop="request_id" label="Request ID" min-width="120" align="center">
          <template #default="{ row }">
            <el-tooltip v-if="row.request_id" :content="row.request_id" placement="top">
              <span class="copyable" @click="copyText(row.request_id)">
                {{ row.request_id.slice(0, 8) }}...
              </span>
            </el-tooltip>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="client_ip" label="客户端IP" width="140" align="center" />
      </el-table>
    </div>

    <div class="pagination-wrapper">
      <el-pagination
        v-model:current-page="query.page"
        v-model:page-size="query.page_size"
        layout="total, prev, pager, next, sizes"
        :total="total"
        :page-sizes="[10, 20, 50, 100]"
        @current-change="load"
        @size-change="load"
      />
    </div>
  </div>
</template>

<style scoped>
.security-audit {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 16px;
}
.filter-form {
  flex-shrink: 0;
  margin-bottom: 12px;
}
.filter-form :deep(.el-form-item) {
  margin-bottom: 8px;
}
.page-error {
  flex-shrink: 0;
  margin-bottom: 12px;
}
.table-wrapper {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
.copyable {
  cursor: pointer;
  color: var(--el-color-primary);
  font-family: monospace;
}
.copyable:hover {
  text-decoration: underline;
}
.pagination-wrapper {
  flex-shrink: 0;
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}
</style>
