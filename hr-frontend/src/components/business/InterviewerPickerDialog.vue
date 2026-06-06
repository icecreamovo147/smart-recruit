<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { listInterviewers } from '@/api/interview'
import type { StaffUserInfo } from '@/types/domain'

const props = defineProps<{
  visible: boolean
  selectedId?: number
  title?: string
  emptyText?: string
  selectLabel?: string
}>()

const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
  (e: 'select', user: StaffUserInfo): void
}>()

const loading = ref(false)
const list = ref<StaffUserInfo[]>([])
const total = ref(0)
const query = reactive({
  page: 1,
  page_size: 10,
  keyword: '',
})

const load = async () => {
  loading.value = true
  try {
    const data = await listInterviewers({
      page: query.page,
      page_size: query.page_size,
      keyword: query.keyword.trim() || undefined,
    })
    list.value = data.list || []
    total.value = Number(data.total) || 0
  } catch (error: unknown) {
    ElMessage.error(error instanceof Error ? error.message : '面试官列表加载失败')
  } finally {
    loading.value = false
  }
}

const search = () => {
  query.page = 1
  load()
}

const handleSelect = (user: StaffUserInfo) => {
  emit('select', user)
  emit('update:visible', false)
}

watch(
  () => props.visible,
  (visible) => {
    if (visible) load()
  },
)

onMounted(() => {
  if (props.visible) load()
})
</script>

<template>
  <el-dialog
    :model-value="props.visible"
    :title="props.title || '选择面试官'"
    width="720px"
    :close-on-click-modal="false"
    @close="emit('update:visible', false)"
  >
    <div class="interviewer-picker">
      <div class="interviewer-picker__toolbar">
        <el-input
          v-model="query.keyword"
          clearable
          placeholder="搜索用户名或邮箱"
          @keyup.enter="search"
          @clear="search"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-button type="primary" @click="search">查询</el-button>
      </div>

      <el-table
        v-loading="loading"
        :data="list"
        height="360"
        row-key="user_id"
        :empty-text="props.emptyText || '暂无面试官'"
        highlight-current-row
        @row-dblclick="handleSelect"
      >
        <el-table-column prop="user_id" label="ID" width="90" />
        <el-table-column prop="username" label="姓名" min-width="160" />
        <el-table-column prop="email" label="邮箱" min-width="220">
          <template #default="{ row }">{{ row.email || '-' }}</template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
              {{ row.status === 'active' ? '启用' : row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button
              size="small"
              type="primary"
              plain
              :disabled="Number(row.user_id) === Number(props.selectedId)"
              @click="handleSelect(row)"
            >
              {{ Number(row.user_id) === Number(props.selectedId) ? '已选择' : (props.selectLabel || '选择') }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="query.page"
        v-model:page-size="query.page_size"
        class="interviewer-picker__pagination"
        layout="total, prev, pager, next, sizes"
        :page-sizes="[10, 20, 50]"
        :total="total"
        @current-change="load"
        @size-change="load"
      />
    </div>
  </el-dialog>
</template>

<style scoped>
.interviewer-picker {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.interviewer-picker__toolbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
}

.interviewer-picker__pagination {
  justify-content: flex-end;
}

@media (max-width: 720px) {
  .interviewer-picker__toolbar {
    grid-template-columns: 1fr;
  }
}
</style>
