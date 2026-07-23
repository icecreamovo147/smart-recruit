<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { getJobOptions, listJobs } from '@/api/job'
import type { DepartmentNode, Job, JobQuery, LocationOption } from '@/types/domain'

type SortKey = 'default' | 'title_asc' | 'title_desc' | 'newest'

interface FlatDepartment {
  id: number
  name: string
  full_name: string
  depth: number
  is_active: number
}

const router = useRouter()
const loading = ref(false)
const optionsLoading = ref(false)
const errorMessage = ref('')
const jobs = ref<Job[]>([])
const total = ref(0)
const PAGE_SIZE_OPTIONS = [10, 20, 30, 50] as const
const query = reactive<JobQuery>({ page: 1, page_size: 10, keyword: '' })

/** Taxonomy from DB (via public job-options). */
const locationOptions = ref<LocationOption[]>([])
const departmentTree = ref<DepartmentNode[]>([])
const selectedLocationIds = ref<number[]>([])
const selectedDepartmentIds = ref<number[]>([])
const sortKey = ref<SortKey>('default')

/** Pool for client-side sort paging when sort is non-default. */
const sortPool = ref<Job[]>([])

let searchTimer: ReturnType<typeof setTimeout> | null = null

const toNum = (v: unknown): number => (v != null && v !== '' ? Number(v) : 0)

const normTree = (nodes: DepartmentNode[]): DepartmentNode[] =>
  (nodes || []).map((n) => ({
    ...n,
    id: toNum(n.id),
    parent_id: toNum(n.parent_id),
    is_active: toNum(n.is_active),
    sort_order: toNum(n.sort_order),
    depth: toNum(n.depth) || 1,
    children: normTree(n.children || []),
  }))

const normLocs = (locs: LocationOption[]): LocationOption[] =>
  (locs || []).map((l) => ({
    ...l,
    id: toNum(l.id),
    is_active: toNum(l.is_active),
    sort_order: toNum(l.sort_order),
  }))

/** Flatten tree in DFS order for hierarchical checkbox rendering. */
const flattenDepartments = (nodes: DepartmentNode[], depth = 0): FlatDepartment[] => {
  const out: FlatDepartment[] = []
  for (const n of nodes) {
    out.push({
      id: n.id,
      name: n.name,
      full_name: n.full_name || n.name,
      depth,
      is_active: n.is_active,
    })
    if (n.children?.length) {
      out.push(...flattenDepartments(n.children, depth + 1))
    }
  }
  return out
}

const flatDepartments = computed(() => flattenDepartments(departmentTree.value))

const hasFacetFilter = computed(
  () => selectedLocationIds.value.length > 0 || selectedDepartmentIds.value.length > 0,
)

const findDepartment = (nodes: DepartmentNode[], id: number): DepartmentNode | null => {
  for (const n of nodes) {
    if (n.id === id) return n
    if (n.children?.length) {
      const hit = findDepartment(n.children, id)
      if (hit) return hit
    }
  }
  return null
}

/** All descendant ids under a node (not including self). */
const collectDescendantIds = (node: DepartmentNode): number[] => {
  const ids: number[] = []
  const walk = (n: DepartmentNode) => {
    for (const c of n.children || []) {
      ids.push(c.id)
      walk(c)
    }
  }
  walk(node)
  return ids
}

/** Cascade checkbox: check/uncheck parent propagates to all children. */
const onDepartmentCheckChange = (raw: Array<string | number>) => {
  const next = raw.map((v) => toNum(v)).filter((id) => id > 0)
  const prev = selectedDepartmentIds.value
  const prevSet = new Set(prev)
  const nextSet = new Set(next)

  const added = next.filter((id) => !prevSet.has(id))
  const removed = prev.filter((id) => !nextSet.has(id))

  const result = new Set(next)

  for (const id of added) {
    const node = findDepartment(departmentTree.value, id)
    if (!node) continue
    for (const childId of collectDescendantIds(node)) {
      result.add(childId)
    }
  }

  for (const id of removed) {
    const node = findDepartment(departmentTree.value, id)
    if (node) {
      for (const childId of collectDescendantIds(node)) {
        result.delete(childId)
      }
      // Uncheck ancestors when any descendant is unchecked.
      let parentId = node.parent_id
      while (parentId > 0) {
        result.delete(parentId)
        const parent = findDepartment(departmentTree.value, parentId)
        parentId = parent?.parent_id ?? 0
      }
    }
  }

  selectedDepartmentIds.value = Array.from(result)
  query.page = 1
  void load()
}

const onLocationCheckChange = (raw: Array<string | number>) => {
  selectedLocationIds.value = raw.map((v) => toNum(v)).filter((id) => id > 0)
  query.page = 1
  void load()
}

const sortJobs = (list: Job[]): Job[] => {
  const next = [...list]
  switch (sortKey.value) {
    case 'title_asc':
      return next.sort((a, b) => (a.title || '').localeCompare(b.title || '', 'zh-CN'))
    case 'title_desc':
      return next.sort((a, b) => (b.title || '').localeCompare(a.title || '', 'zh-CN'))
    case 'newest':
      return next.sort((a, b) => {
        const ta = Date.parse(a.created_at || a.createdAt || '') || 0
        const tb = Date.parse(b.created_at || b.createdAt || '') || 0
        return tb - ta
      })
    default:
      return next
  }
}

const applySortPipeline = (source: Job[]) => {
  const sorted = sortJobs(source)
  total.value = sorted.length
  const page = Number(query.page) || 1
  const size = Number(query.page_size) || 10
  const start = (page - 1) * size
  jobs.value = sorted.slice(start, start + size)
}

const buildListParams = (page: number, pageSize: number): JobQuery => {
  const params: JobQuery = {
    page,
    page_size: pageSize,
    keyword: query.keyword || undefined,
  }
  if (selectedDepartmentIds.value.length > 0) {
    params.department_ids = [...selectedDepartmentIds.value]
  }
  if (selectedLocationIds.value.length > 0) {
    params.location_ids = [...selectedLocationIds.value]
  }
  return params
}

const loadOptions = async () => {
  optionsLoading.value = true
  try {
    const res = await getJobOptions()
    departmentTree.value = normTree(res.department_tree || [])
    locationOptions.value = normLocs(res.locations || []).filter((l) => l.is_active !== 0)
  } catch {
    // Keep filters empty; job list can still load without taxonomy.
  } finally {
    optionsLoading.value = false
  }
}

const load = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    query.page = Number(query.page) || 1
    query.page_size = Number(query.page_size) || 10

    // Non-default sort: pull a larger server-filtered pool and sort client-side.
    if (sortKey.value !== 'default') {
      const data = await listJobs(buildListParams(1, 100))
      const list = data.list || []
      sortPool.value = list
      applySortPipeline(list)
      return
    }

    const data = await listJobs(buildListParams(query.page, query.page_size))
    sortPool.value = []
    jobs.value = data.list || []
    total.value = Number(data.total) || 0
  } catch (error: unknown) {
    errorMessage.value = error instanceof Error ? error.message : '岗位列表加载失败'
  } finally {
    loading.value = false
  }
}

const search = () => {
  query.page = 1
  void load()
}

const clearFacets = () => {
  selectedLocationIds.value = []
  selectedDepartmentIds.value = []
  query.page = 1
  void load()
}

const resultLabel = computed(() => `共 ${total.value} 个在招岗位`)

watch(
  () => query.keyword,
  () => {
    query.page = 1
    if (searchTimer) clearTimeout(searchTimer)
    searchTimer = setTimeout(() => {
      void load()
    }, 500)
  },
)

// Sort changes reload list (location/department handlers call load directly).
watch(sortKey, () => {
  query.page = 1
  void load()
})

const onPageChange = (page: number) => {
  query.page = page
  if (sortKey.value !== 'default' && sortPool.value.length > 0) {
    applySortPipeline(sortPool.value)
    return
  }
  void load()
}

const onPageSizeChange = (size: number) => {
  query.page_size = Number(size) || 10
  query.page = 1
  // Non-default sort re-slices the existing pool; otherwise refetch from server.
  if (sortKey.value !== 'default' && sortPool.value.length > 0) {
    applySortPipeline(sortPool.value)
    return
  }
  void load()
}

const openJob = (jobId: number) => {
  void router.push(`/jobs/${jobId}`)
}

onMounted(async () => {
  await loadOptions()
  await load()
})
</script>

<template>
  <section class="job-list-page">
    <!-- Single surface: page header + filter/results board -->
    <div class="job-shell" v-loading="loading">
      <header class="workspace-header">
        <div>
          <span class="workspace-eyebrow">OPEN POSITIONS</span>
          <h1 class="page-title">正在招聘</h1>
          <p class="page-subtitle">发现最适合你的岗位与团队。</p>
        </div>
      </header>

      <div class="job-board">
        <aside class="job-board__filter" aria-label="筛选条件">
          <el-scrollbar class="job-board__filter-scroll">
            <div class="job-filter">
              <div class="job-filter__head">
                <h2 class="job-filter__title">筛选条件</h2>
                <button
                  v-if="hasFacetFilter"
                  type="button"
                  class="job-filter__clear"
                  @click="clearFacets"
                >
                  清空
                </button>
              </div>

              <div class="job-filter__body" v-loading="optionsLoading">
                <div class="job-filter__group">
                  <h3 class="job-filter__group-title">工作地点</h3>
                  <el-checkbox-group
                    :model-value="selectedLocationIds"
                    class="job-filter__checks"
                    @change="onLocationCheckChange"
                  >
                    <el-checkbox
                      v-for="loc in locationOptions"
                      :key="loc.id"
                      :value="loc.id"
                    >
                      {{ loc.name }}
                    </el-checkbox>
                  </el-checkbox-group>
                  <p v-if="!optionsLoading && locationOptions.length === 0" class="job-filter__empty">
                    暂无可选地点
                  </p>
                </div>

                <div class="job-filter__group">
                  <h3 class="job-filter__group-title">所属部门</h3>
                  <el-checkbox-group
                    :model-value="selectedDepartmentIds"
                    class="job-filter__checks job-filter__checks--tree"
                    @change="onDepartmentCheckChange"
                  >
                    <el-checkbox
                      v-for="dept in flatDepartments"
                      :key="dept.id"
                      :value="dept.id"
                      class="job-filter__dept"
                      :style="{ paddingLeft: `${dept.depth * 14}px` }"
                      :title="dept.full_name"
                    >
                      {{ dept.name }}
                    </el-checkbox>
                  </el-checkbox-group>
                  <p v-if="!optionsLoading && flatDepartments.length === 0" class="job-filter__empty">
                    暂无可选部门
                  </p>
                </div>
              </div>
            </div>
          </el-scrollbar>
        </aside>

        <div class="job-board__divider" aria-hidden="true" />

        <div class="job-board__main">
          <div class="job-board__toolbar">
            <p class="job-board__count">{{ resultLabel }}</p>

            <div class="job-board__controls">
              <el-input
                v-model="query.keyword"
                class="job-board__search-input"
                clearable
                placeholder="搜索岗位、部门、地点"
                @keyup.enter="search"
              />
              <el-button
                type="primary"
                class="job-board__search-btn"
                :loading="loading"
                @click="search"
              >
                搜索
              </el-button>
              <el-select
                v-model="sortKey"
                class="job-board__sort"
                placeholder="默认排序"
              >
                <el-option label="默认排序" value="default" />
                <el-option label="最新发布" value="newest" />
                <el-option label="职位名称 A-Z" value="title_asc" />
                <el-option label="职位名称 Z-A" value="title_desc" />
              </el-select>
            </div>
          </div>

          <el-alert
            v-if="errorMessage"
            class="job-board__error"
            type="error"
            :title="errorMessage"
            show-icon
            :closable="false"
          >
            <template #default>
              <el-button size="small" type="danger" plain @click="load">重试</el-button>
            </template>
          </el-alert>

          <el-scrollbar class="job-board__content">
            <el-empty
              v-if="!loading && !errorMessage && jobs.length === 0"
              class="job-board__empty"
              :description="hasFacetFilter || query.keyword ? '暂无数据' : '暂无在招岗位'"
            />

            <div v-else-if="jobs.length > 0" class="job-grid">
              <article
                v-for="job in jobs"
                :key="job.job_id"
                class="job-card"
              >
                <div class="job-card__top">
                  <h2 class="job-card__title" :title="job.title">{{ job.title }}</h2>
                  <span class="job-card__salary">
                    {{ job.salary_range ? `${job.salary_range} 元/月` : '薪资面议' }}
                  </span>
                </div>

                <p
                  class="job-card__meta"
                  :title="`${job.department || '未填写部门'} · ${job.location || '地点待定'}`"
                >
                  <span class="job-card__meta-text">
                    {{ job.department || '未填写部门' }} · {{ job.location || '地点待定' }}
                  </span>
                </p>

                <div class="job-card__footer">
                  <el-tag
                    :type="job.status === 1 ? 'success' : 'info'"
                    size="small"
                    effect="plain"
                    class="job-card__status"
                  >
                    {{ job.status === 1 ? '招聘中' : '已下架' }}
                  </el-tag>
                  <el-button type="primary" size="small" class="job-card__action" @click="openJob(job.job_id)">
                    查看详情
                  </el-button>
                </div>
              </article>
            </div>
          </el-scrollbar>

          <div class="job-board__pagination">
            <el-pagination
              v-model:current-page="query.page"
              v-model:page-size="query.page_size"
              layout="total, sizes, prev, pager, next, jumper"
              :total="total"
              :page-sizes="[...PAGE_SIZE_OPTIONS]"
              background
              @current-change="onPageChange"
              @size-change="onPageSizeChange"
            />
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.job-list-page {
  display: flex;
  flex-direction: column;
  gap: 0;
  width: 100%;
  height: 100%;
  min-height: 0;
}

/* ── Single shell: header + body (fills remaining page height) ──── */
.job-shell {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 16px;
  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.07);
  overflow: hidden;
}

/* ── Filter + results body ──────────────────────────────────────── */
.job-board {
  flex: 1 1 auto;
  min-height: 0;
  display: grid;
  grid-template-columns: 236px 1px minmax(0, 1fr);
  grid-template-rows: minmax(0, 1fr);
  align-items: stretch;
  overflow: hidden;
}

.job-board__divider {
  width: 1px;
  background: var(--border);
  align-self: stretch;
}

/* ── Filter pane (no independent card chrome) ───────────────────── */
.job-board__filter {
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  background: transparent;
}

.job-board__filter-scroll {
  height: 100%;
}

.job-board__filter-scroll :deep(.el-scrollbar__wrap) {
  overflow-x: hidden;
}

.job-filter {
  padding: 22px 18px 24px;
}

.job-filter__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 18px;
}

.job-filter__title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
}

.job-filter__clear {
  border: none;
  background: transparent;
  color: var(--brand);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  padding: 2px 4px;
  border-radius: 6px;
}

.job-filter__clear:hover {
  color: var(--brand-strong);
  background: var(--brand-soft);
}

.job-filter__group + .job-filter__group {
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px solid var(--border);
}

.job-filter__group-title {
  margin: 0 0 12px;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.02em;
  color: var(--text-secondary);
}

.job-filter__checks {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 10px;
  width: 100%;
}

.job-filter__checks :deep(.el-checkbox) {
  display: flex;
  align-items: flex-start;
  margin-right: 0;
  height: auto;
  width: 100%;
  white-space: normal;
}

.job-filter__checks :deep(.el-checkbox__input) {
  margin-top: 2px;
  flex-shrink: 0;
}

.job-filter__checks :deep(.el-checkbox__label) {
  color: var(--text-primary);
  font-size: 14px;
  line-height: 1.45;
  white-space: normal;
  word-break: break-word;
  padding-left: 8px;
}

.job-filter__checks--tree {
  gap: 6px;
}

.job-filter__dept {
  width: 100%;
  box-sizing: border-box;
}

.job-filter__empty {
  margin: 0;
  font-size: 13px;
  color: var(--text-muted);
}

/* ── Results pane ───────────────────────────────────────────────── */
.job-board__main {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 20px 22px 22px;
  overflow: hidden;
}

.job-board__toolbar {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px 18px;
  flex-wrap: wrap;
  margin-bottom: 18px;
}

.job-board__count {
  margin: 0;
  flex: 0 0 auto;
  font-size: 15px;
  font-weight: 650;
  color: var(--text-primary);
  white-space: nowrap;
}

.job-board__controls {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1 1 320px;
  min-width: 0;
  justify-content: flex-end;
  flex-wrap: wrap;
}

.job-board__search-input {
  flex: 1 1 280px;
  min-width: 0;
  max-width: 360px;
}

.job-board__search-btn {
  flex: 0 0 auto;
}

.job-board__sort {
  width: 140px;
  flex: 0 0 auto;
}

.job-board__error {
  flex: 0 0 auto;
  margin-bottom: 16px;
}

/* Element Plus scrollbar fills the flex middle region */
.job-board__content {
  flex: 1 1 0;
  min-height: 0;
  height: 100%;
}

.job-board__content :deep(.el-scrollbar__wrap) {
  /* Hide native bar; el-scrollbar draws its own */
  overflow-x: hidden;
}

.job-board__content :deep(.el-scrollbar__view) {
  min-height: 100%;
  box-sizing: border-box;
  /* Small end padding so last row isn’t flush against the bar */
  padding-right: 4px;
  padding-bottom: 2px;
}

.job-board__empty {
  padding: 48px 0 24px;
}

/* ── Job grid: 3 columns by default ─────────────────────────────── */
.job-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 18px;
  width: 100%;
  align-content: start;
}

/* ── Job card ───────────────────────────────────────────────────── */
.job-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
  min-height: 100%;
  padding: 20px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 14px;
  box-shadow: none;
  transition: border-color var(--motion-normal) var(--motion-ease);
}

.job-card:hover {
  /* Keep a light border cue only — no lift / no shadow */
  border-color: color-mix(in srgb, var(--brand) 35%, var(--border));
}

.job-card__top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  min-width: 0;
}

.job-card__title {
  margin: 0;
  min-width: 0;
  flex: 1 1 auto;
  font-size: 17px;
  font-weight: 700;
  line-height: 1.4;
  color: var(--text-primary);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-word;
}

.job-card__salary {
  flex: 0 0 auto;
  max-width: 42%;
  padding: 4px 10px;
  border-radius: 999px;
  background: var(--brand-soft);
  color: var(--brand);
  font-size: 12px;
  font-weight: 600;
  line-height: 1.4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.job-card__meta {
  margin: 0;
  min-width: 0;
  color: var(--text-muted);
  font-size: 13.5px;
  line-height: 1.45;
}

.job-card__meta-text {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-word;
}

.job-card__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: auto;
  padding-top: 4px;
}

.job-card__status {
  flex: 0 0 auto;
  opacity: 0.92;
}

.job-card__action {
  font-weight: 600;
}

/* ── Pagination: pinned to bottom of board ──────────────────────── */
.job-board__pagination {
  flex: 0 0 auto;
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  margin-top: 20px;
  padding-top: 4px;
}

.job-board__pagination :deep(.el-pagination) {
  flex-wrap: wrap;
  justify-content: flex-end;
  row-gap: 8px;
}

/* ── Responsive ─────────────────────────────────────────────────── */

/* Narrow laptop: keep 2-col, slightly tighter filter */
@media (max-width: 1200px) {
  .job-board {
    grid-template-columns: 220px 1px minmax(0, 1fr);
  }

  .job-board__main {
    padding: 18px 18px 20px;
  }

  .job-filter {
    padding: 18px 14px 20px;
  }
}

/* Stack filter above results */
@media (max-width: 960px) {
  .job-board {
    grid-template-columns: 1fr;
    grid-template-rows: auto minmax(0, 1fr);
    overflow: hidden;
  }

  .job-board__divider {
    width: 100%;
    height: 1px;
    grid-column: 1 / -1;
  }

  .job-board__filter {
    max-height: 220px;
  }

  .job-board__filter-scroll {
    height: 100%;
    max-height: 220px;
  }

  .job-filter {
    padding: 16px 18px 14px;
  }

  .job-filter__body {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px 24px;
  }

  .job-filter__group + .job-filter__group {
    margin-top: 0;
    padding-top: 0;
    border-top: none;
    border-left: 1px solid var(--border);
    padding-left: 20px;
  }

  .job-board__main {
    padding-top: 16px;
    min-height: 280px;
  }

  .job-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

/* Tablet / large phone: single column cards */
@media (max-width: 768px) {
  .workspace-header {
    align-items: flex-start;
    padding: 20px 18px 18px;
  }

  .workspace-header .page-title {
    font-size: 26px;
  }

  .job-filter__body {
    grid-template-columns: 1fr;
  }

  .job-filter__group + .job-filter__group {
    margin-top: 14px;
    padding-top: 14px;
    padding-left: 0;
    border-left: none;
    border-top: 1px solid var(--border);
  }

  .job-filter__checks {
    flex-direction: row;
    flex-wrap: wrap;
    gap: 8px 16px;
  }

  .job-filter__checks :deep(.el-checkbox) {
    width: auto;
    max-width: 100%;
  }

  .job-grid {
    grid-template-columns: 1fr;
    gap: 14px;
  }

  .job-board__toolbar {
    align-items: stretch;
  }

  .job-board__controls {
    justify-content: stretch;
    width: 100%;
  }

  .job-board__search-input {
    flex: 1 1 100%;
    max-width: none;
  }

  .job-board__search-btn {
    flex: 1 1 auto;
  }

  .job-board__sort {
    width: 100%;
  }

  .job-board__pagination {
    justify-content: center;
    margin-top: 20px;
  }

  .job-board__pagination :deep(.el-pagination) {
    justify-content: center;
  }
}

@media (max-width: 480px) {
  .job-shell {
    border-radius: 12px;
  }

  .job-board__main {
    padding: 14px 14px 16px;
  }

  .job-filter {
    padding: 14px;
  }

  .job-card {
    padding: 16px;
  }

  .job-card__salary {
    max-width: 46%;
    font-size: 11px;
  }
}
</style>
