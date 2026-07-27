<script setup lang="ts">
import { t } from '@shared/i18n'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown, MoreFilled, Plus, Refresh, Search } from '@element-plus/icons-vue'
import { createJob, getJobOptions, listHRJobs, offlineJob, onlineJob, updateJob } from '@/api/job'
import RichTextEditor from '@/components/RichTextEditor.vue'
import type { DepartmentLocationMapItem, DepartmentNode, Job, JobCreatePayload, JobQuery, LocationOption } from '@/types/domain'

const router = useRouter()
const loading = ref(false)
const errorMessage = ref('')
const saving = ref(false)
const dialogVisible = ref(false)
const editingId = ref<number | null>(null)
const jobs = ref<Job[]>([])
const total = ref(0)
const query = reactive<JobQuery>({ page: 1, page_size: 10 })
const keyword = ref('')
const statusFilter = ref('')

// Taxonomy options loaded from API
const departmentTree = ref<DepartmentNode[]>([])
const locationOptions = ref<LocationOption[]>([])
const deptLocationMap = ref<Record<number, number[]>>({})

// Available locations for the currently selected department
const availableLocations = computed(() => {
  const deptId = toNum(form.department_id)
  if (!deptId) return locationOptions.value
  const ids = deptLocationMap.value[deptId]
  if (!ids || ids.length === 0) return [] as LocationOption[]
  const idSet = new Set(ids)
  const filtered = locationOptions.value.filter((loc) => idSet.has(loc.id))
  // When editing a history job, ensure the current location is included for display
  // even if it's not in the available list (prevents "location vanished" rendering issues).
  if (editingId.value && form.location_id) {
    const curId = toNum(form.location_id)
    if (!idSet.has(curId)) {
      const curLoc = locationOptions.value.find((l) => l.id === curId)
      if (curLoc) {
        return [curLoc, ...filtered]
      }
    }
  }
  return filtered
})

// Whether locations are available for the selected department
const hasNoAvailableLocations = computed(() => {
  const deptId = toNum(form.department_id)
  if (!deptId) return false
  const ids = deptLocationMap.value[deptId]
  return !ids || ids.length === 0
})

const form = reactive<JobCreatePayload>({
  title: '',
  department: '',
  department_id: undefined,
  location: '',
  location_id: undefined,
  salary_range: '',
  description: '',
  requirements: '',
})

// Flatten department tree for tree-select label lookup
const flattenTree = (nodes: DepartmentNode[]): Record<number, string> => {
  const result: Record<number, string> = {}
  const walk = (list: DepartmentNode[]) => {
    for (const n of list) {
      result[n.id] = n.name
      if (n.children?.length) walk(n.children)
    }
  }
  walk(nodes)
  return result
}

// protojson serializes int64 as strings; normalize to numbers for form bindings.
const toNum = (v: unknown): number => (v != null ? Number(v) : 0)
const normTree = (nodes: DepartmentNode[]): DepartmentNode[] =>
  nodes.map(n => ({ ...n, id: toNum(n.id), parent_id: toNum(n.parent_id), children: normTree(n.children || []) }))
const normLocs = (locs: LocationOption[]): LocationOption[] =>
  locs.map(l => ({ ...l, id: toNum(l.id) }))

const loadOptions = async () => {
  try {
    const res = await getJobOptions()
    departmentTree.value = normTree(res.department_tree || [])
    locationOptions.value = normLocs(res.locations || [])
    // Build department → location_ids map
    const map: Record<number, number[]> = {}
    if (res.department_location_map) {
      for (const item of res.department_location_map) {
        map[toNum(item.department_id)] = (item.location_ids || []).map(toNum)
      }
    }
    deptLocationMap.value = map
  } catch {
    // silently keep empty
  }
}

const escapeHtml = (value: string = ''): string =>
  String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')

const hasHtml = (value: string = ''): boolean => /<\/?[a-z][\s\S]*>/i.test(value)

const normalizeRichText = (value: string = ''): string => {
  const text = String(value || '').trim()
  if (!text || hasHtml(text)) return text
  return escapeHtml(text)
    .split(/\n{2,}/)
    .map((block) => `<p>${block.replace(/\n/g, '<br>')}</p>`)
    .join('')
}

// Guard to suppress watch-triggered clearing during form population (e.g. openEdit).
const populating = ref(false)

// Clear location when department changes (user interaction only).
// flush: 'sync' so the guard flag is still true during Object.assign in openEdit.
watch(() => form.department_id, () => {
  if (populating.value) return
  form.location_id = undefined
  form.location = ''
}, { flush: 'sync' })

const resetForm = () => {
  editingId.value = null
  Object.assign(form, { title: '', department: '', department_id: undefined, location: '', location_id: undefined, salary_range: '', description: '', requirements: '' })
}

const load = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    query.page = Number(query.page) || 1
    query.page_size = Number(query.page_size) || 10
    const data = await listHRJobs(query)
    jobs.value = data.list || []
    total.value = Number(data.total) || 0
  } catch (error: unknown) {
    errorMessage.value = error instanceof Error ? error.message : '岗位列表加载失败'
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  resetForm()
  dialogVisible.value = true
}

const openCopy = (row: Job) => {
  editingId.value = null
  populating.value = true
  Object.assign(form, {
    title: row.title,
    department: row.department,
    department_id: toNum(row.department_id ?? row.departmentId),
    location: row.location,
    location_id: toNum(row.location_id ?? row.locationId),
    salary_range: row.salary_range,
    description: normalizeRichText(row.description),
    requirements: normalizeRichText(row.requirements),
  })
  populating.value = false
  dialogVisible.value = true
}

const openEdit = (row: Job) => {
  editingId.value = row.job_id
  populating.value = true
  Object.assign(form, {
    title: row.title,
    department: row.department,
    department_id: toNum(row.department_id ?? row.departmentId),
    location: row.location,
    location_id: toNum(row.location_id ?? row.locationId),
    salary_range: row.salary_range,
    description: normalizeRichText(row.description),
    requirements: normalizeRichText(row.requirements),
  })
  populating.value = false
  dialogVisible.value = true
}

const save = async () => {
  if (!form.title.trim()) {
    ElMessage.warning(t('common.invalid_request'))
    return
  }
  if (!form.department_id && !form.department) {
    ElMessage.warning(t('common.invalid_request'))
    return
  }
  if (!form.location_id && !form.location) {
    ElMessage.warning(t('common.invalid_request'))
    return
  }
  saving.value = true
  try {
    // Ensure IDs are numbers (protojson may return strings for int64)
    const payload = {
      ...form,
      department_id: toNum(form.department_id),
      location_id: toNum(form.location_id),
    }
    if (editingId.value) {
      await updateJob(editingId.value, payload)
      ElMessage.success(t('common.success'))
    } else {
      await createJob(payload)
      ElMessage.success(t('common.success'))
    }
    dialogVisible.value = false
    load()
  } finally {
    saving.value = false
  }
}

const offline = async (row: Job) => {
  try {
    await ElMessageBox.confirm(`确认下架「${row.title}」？`, '下架岗位', { type: 'warning' })
  } catch {
    return
  }
  try {
    await offlineJob(row.job_id)
    ElMessage.success(t('common.success'))
    load()
  } catch {
    // error already shown by request interceptor
  }
}

const online = async (row: Job) => {
  try {
    await ElMessageBox.confirm(`确认重新上线「${row.title}」？`, '上线岗位', { type: 'info' })
  } catch {
    return
  }
  try {
    await onlineJob(row.job_id)
    ElMessage.success(t('common.success'))
    load()
  } catch {
    // error already shown by request interceptor
  }
}

const filteredJobs = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return jobs.value.filter((item) => {
    const matchesKeyword = !q
      || item.title.toLowerCase().includes(q)
      || (item.department || '').toLowerCase().includes(q)
      || (item.location || '').toLowerCase().includes(q)
    const matchesStatus = !statusFilter.value
      || (statusFilter.value === 'online' ? item.status === 1 : item.status !== 1)
    return matchesKeyword && matchesStatus
  })
})

onMounted(() => {
  load()
  loadOptions()
})
</script>

<template>
  <section class="console-page console-page--fill job-management-page">
    <div class="job-management-workspace">
      <div class="workspace-header">
        <div class="workspace-header__copy">
          <p class="console-eyebrow">RECRUITING OPS</p>
          <h1 class="console-title">岗位管理</h1>
          <p class="console-description">管理招聘岗位的发布状态、部门地点、薪资信息和候选人台账入口，让岗位运营状态一眼可见。</p>
        </div>
      </div>

      <div class="workspace-divider"></div>

      <div class="workspace-toolbar">
        <div class="workspace-toolbar__filters">
          <el-input v-model="keyword" :prefix-icon="Search" clearable placeholder="搜索岗位 / 部门 / 地点" style="width: 260px" />
          <el-select v-model="statusFilter" clearable placeholder="全部状态" style="width: 140px">
            <el-option label="招募中" value="online" />
            <el-option label="已下架" value="offline" />
          </el-select>
        </div>
        <div class="workspace-toolbar__actions">
          <el-button :icon="Refresh" @click="load">刷新</el-button>
          <el-button type="primary" :icon="Plus" @click="openCreate">新增岗位</el-button>
        </div>
      </div>

      <el-alert v-if="errorMessage" class="workspace-error" type="error" :title="errorMessage" show-icon :closable="false">
        <template #default>
          <el-button size="small" type="danger" plain @click="load">重试</el-button>
        </template>
      </el-alert>

      <div class="workspace-table-area desktop-only">
        <el-table class="workspace-table" height="100%" v-loading="loading" :data="filteredJobs" empty-text="暂无岗位">
          <el-table-column label="岗位信息" min-width="240">
            <template #default="{ row }">
              <div class="console-entity">
                <div class="console-entity__name">{{ row.title }}</div>
                <div class="console-entity__meta">{{ row.department || '未填写部门' }} / {{ row.location || '地点待定' }}</div>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="salary_range" label="薪资" width="140" align="center" />
          <el-table-column label="投递数" width="90" align="center">
            <template #default="{ row }">
              {{ row.application_count ?? 0 }}
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '招募中' : '已下架' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200" fixed="right" align="center">
            <template #default="{ row }">
              <el-button size="small" @click="openEdit(row)">编辑</el-button>
              <el-dropdown trigger="click" @command="(cmd: string) => { if (cmd === 'copy') openCopy(row); if (cmd === 'apps') router.push(`/hr/jobs/${row.job_id}/applications`); if (cmd === 'offline') offline(row); if (cmd === 'online') online(row) }">
                <el-button size="small">更多<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="copy">复制</el-dropdown-item>
                    <el-dropdown-item command="apps">台账</el-dropdown-item>
                    <el-dropdown-item v-if="row.status === 1" command="offline" divided style="color: var(--el-color-danger)">下架</el-dropdown-item>
                    <el-dropdown-item v-else command="online" divided>上线</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div class="workspace-pagination desktop-only">
        <el-pagination v-model:current-page="query.page" v-model:page-size="query.page_size" layout="total, prev, pager, next, sizes" :total="total" @current-change="load" @size-change="load" />
      </div>

      <div class="mobile-card-list mobile-only">
        <el-empty v-if="!loading && filteredJobs.length === 0" description="暂无岗位" />
        <div v-for="job in filteredJobs" :key="job.job_id" class="mobile-job-card">
          <div class="mobile-card__header">
            <h3 class="mobile-card__title">{{ job.title }}</h3>
            <el-tag :type="job.status === 1 ? 'success' : 'info'" size="small">{{ job.status === 1 ? '招募中' : '已下架' }}</el-tag>
          </div>
          <div class="mobile-card__meta">
            <span>{{ job.department || '未填写部门' }}</span>
            <span>{{ job.location || '地点待定' }}</span>
            <span>{{ job.salary_range ? job.salary_range + ' 元/月' : '薪资面议' }}</span>
            <span>投递 {{ job.application_count ?? 0 }} 人</span>
          </div>
          <div class="mobile-card__actions">
            <el-button size="small" type="primary" plain @click="openEdit(job)">编辑</el-button>
            <el-button size="small" type="primary" plain @click="openCopy(job)">复制</el-button>
            <el-button size="small" type="primary" plain @click="router.push(`/hr/jobs/${job.job_id}/applications`)">台账</el-button>
            <el-button v-if="job.status === 1" size="small" type="danger" plain @click="offline(job)">下架</el-button>
            <el-button v-else size="small" type="success" plain @click="online(job)">上线</el-button>
          </div>
        </div>
      </div>
    </div>

    <el-drawer v-model="dialogVisible" :title="editingId ? '编辑岗位' : '新增岗位'" size="860px" :close-on-click-modal="true" @closed="resetForm">
      <el-form label-width="80px">
        <el-form-item label="岗位名称">
          <el-input v-model="form.title" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :xs="24" :sm="8">
            <el-form-item label="部门">
              <el-tree-select
                v-model="form.department_id"
                :data="departmentTree"
                :props="{ label: 'name', value: 'id', children: 'children' }"
                class="job-form-select"
                filterable
                check-strictly
                placeholder="选择部门"
                node-key="id"
              />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="8">
            <el-form-item label="地点">
              <el-select
                v-model="form.location_id"
                class="job-form-select"
                filterable
                placeholder="选择地点"
                :disabled="hasNoAvailableLocations"
              >
                <el-option v-for="item in availableLocations" :key="item.id" :label="item.name" :value="item.id" />
              </el-select>
              <span v-if="hasNoAvailableLocations" class="location-hint">该部门暂无可选地点</span>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="8"><el-form-item label="薪资"><el-input v-model="form.salary_range"><template #suffix><span class="salary-unit-suffix">元/月</span></template></el-input></el-form-item></el-col>
        </el-row>
        <el-form-item label="岗位描述">
          <RichTextEditor v-model="form.description" />
        </el-form-item>
        <el-form-item label="任职要求">
          <RichTextEditor v-model="form.requirements" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-drawer>
  </section>
</template>

<style scoped>
.job-management-workspace {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 10px;
  box-shadow: var(--admin-console-card-shadow);
  overflow: hidden;
}

.workspace-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  padding: 22px 24px;
  background: var(--admin-console-header-bg);
  flex-shrink: 0;
}

.workspace-header__copy {
  min-width: 0;
}

.workspace-divider {
  height: 1px;
  background: var(--border);
  flex-shrink: 0;
}

.workspace-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 24px;
  flex-shrink: 0;
}

.workspace-toolbar__filters {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.workspace-toolbar__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.workspace-error {
  margin: 0 24px 12px;
  flex-shrink: 0;
}

.workspace-table-area {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 0 10px;
}

.workspace-table-area > .el-table {
  height: 100%;
  min-height: 320px;
}

.workspace-pagination {
  display: flex;
  justify-content: flex-end;
  padding: 12px 24px 14px;
  border-top: 1px solid var(--border);
  flex-shrink: 0;
}

@media (max-width: 720px) {
  .workspace-header {
    flex-direction: column;
  }
  .workspace-toolbar {
    flex-direction: column;
    align-items: stretch;
  }
  .workspace-toolbar__actions {
    justify-content: flex-start;
  }
}
</style>
