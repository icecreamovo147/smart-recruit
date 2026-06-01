<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
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
    ElMessage.warning('请填写岗位名称')
    return
  }
  if (!form.department_id && !form.department) {
    ElMessage.warning('请选择部门')
    return
  }
  if (!form.location_id && !form.location) {
    ElMessage.warning('请选择地点')
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
      ElMessage.success('岗位已更新')
    } else {
      await createJob(payload)
      ElMessage.success('岗位已创建')
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
    ElMessage.success('岗位已下架')
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
    ElMessage.success('岗位已上线')
    load()
  } catch {
    // error already shown by request interceptor
  }
}

onMounted(() => {
  load()
  loadOptions()
})
</script>

<template>
  <section class="job-management-page">
    <div class="page-header">
      <h1 class="page-title">岗位管理</h1>
      <div class="toolbar">
        <el-button type="primary" @click="openCreate">新增岗位</el-button>
      </div>
    </div>

    <div class="content-surface job-management-surface">
      <el-alert v-if="errorMessage" class="page-error" type="error" :title="errorMessage" show-icon :closable="false">
        <template #default>
          <el-button size="small" type="danger" plain @click="load">重试</el-button>
        </template>
      </el-alert>
      <div class="job-table-area desktop-only">
        <el-table class="job-table" height="100%" v-loading="loading" :data="jobs" empty-text="暂无岗位">
          <el-table-column prop="title" label="岗位" min-width="160" align="center" />
          <el-table-column prop="department" label="部门" width="120" align="center" />
          <el-table-column prop="location" label="地点" width="140" align="center" />
          <el-table-column prop="salary_range" label="薪资" width="130" align="center" />
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
          <el-table-column label="操作" width="280" fixed="right" align="center">
            <template #default="{ row }">
              <el-button text type="primary" @click="openEdit(row)">编辑</el-button>
              <el-button text type="primary" @click="router.push(`/hr/jobs/${row.job_id}/applications`)">台账</el-button>
              <el-button v-if="row.status === 1" text type="danger" @click="offline(row)">下架</el-button>
              <el-button v-else text type="success" @click="online(row)">上线</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <!-- Mobile job cards -->
      <div class="mobile-card-list mobile-only">
        <el-empty v-if="!loading && jobs.length === 0" description="暂无岗位" />
        <div v-for="job in jobs" :key="job.job_id" class="mobile-job-card">
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
            <el-button size="small" type="primary" plain @click="router.push(`/hr/jobs/${job.job_id}/applications`)">台账</el-button>
            <el-button v-if="job.status === 1" size="small" type="danger" plain @click="offline(job)">下架</el-button>
            <el-button v-else size="small" type="success" plain @click="online(job)">上线</el-button>
          </div>
        </div>
      </div>
      <el-pagination class="job-pagination" v-model:current-page="query.page" v-model:page-size="query.page_size" layout="total, prev, pager, next, sizes" :total="total" @current-change="load" @size-change="load" />
    </div>

    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑岗位' : '新增岗位'" width="860px" @closed="resetForm">
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
    </el-dialog>
  </section>
</template>
