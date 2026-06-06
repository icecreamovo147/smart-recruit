<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft, Briefcase, Clock, Document, Plus, TrendCharts, User } from '@element-plus/icons-vue'
import { getCandidateWorkspace, createNote, listNotes, createTag, listTags, assignTag, unassignTag, createFollowUpTask, listFollowUpTasks, completeFollowUpTask } from '@/api/collaboration'
import { getHRStatusLabel, getStatusType } from '@/types/domain'
import type { CandidateWorkspace, CandidateNoteInfo, CandidateTagInfo, FollowUpTaskInfo, StaffUserInfo } from '@/types/domain'
import { renderRichText } from '@/utils/richText'
import TimelineView from '@/components/business/TimelineView.vue'
import InterviewerPickerDialog from '@/components/business/InterviewerPickerDialog.vue'

const route = useRoute()
const router = useRouter()

const candidateUserId = Number(route.params.candidateUserId)
const sectionOptions = [
  { key: 'overview', label: '概览' },
  { key: 'applications', label: '投递记录' },
  { key: 'process', label: '面试 & Offer' },
  { key: 'collaboration', label: '协作' },
  { key: 'timeline', label: '动态时间线' },
] as const
const sectionKeys = new Set(sectionOptions.map((item) => item.key))

const loading = ref(false)
const workspace = ref<CandidateWorkspace | null>(null)
const collaborationLoaded = ref(false)

const notes = ref<CandidateNoteInfo[]>([])
const newNoteContent = ref('')
const noteLoading = ref(false)

const allTags = ref<CandidateTagInfo[]>([])
const tagInput = ref('')
const tagLoading = ref(false)

const tasks = ref<FollowUpTaskInfo[]>([])
const taskLoading = ref(false)
const taskDialogVisible = ref(false)
const assigneePickerVisible = ref(false)
const selectedAssigneeName = ref('')
const newTask = ref({
  title: '',
  description: '',
  assignee_user_id: 0,
  due_at: '',
})

const formatDateTime = (value: string): string => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (num: number): string => String(num).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

const activeSection = computed(() => {
  const section = String(route.params.section || 'overview')
  return sectionKeys.has(section as typeof sectionOptions[number]['key']) ? section : 'overview'
})

const candidateTags = computed<CandidateTagInfo[]>(() => workspace.value?.tags || [])
const availableTags = computed(() => {
  const assigned = new Set(candidateTags.value.map((tag) => Number(tag.id)))
  return allTags.value.filter((tag) => !assigned.has(Number(tag.id)))
})
const workExperienceHtml = computed(() => renderRichText(workspace.value?.work_experience || '', '-'))
const toNumber = (value: unknown): number => {
  const n = Number(value ?? 0)
  return Number.isFinite(n) ? n : 0
}

const overviewStats = computed(() => {
  const source = workspace.value as (CandidateWorkspace & {
    totalApplications?: number
    totalInterviews?: number
    totalOffers?: number
  }) | null
  return {
    applications: toNumber(source?.total_applications ?? source?.totalApplications),
    interviews: toNumber(source?.total_interviews ?? source?.totalInterviews),
    offers: toNumber(source?.total_offers ?? source?.totalOffers),
  }
})

const currentApplication = computed(() => {
  const applications = workspace.value?.applications || []
  return applications.find((app) => Number(app.is_current) === 1) || applications[0] || null
})

const recentApplications = computed(() => (workspace.value?.applications || []).slice(0, 3))
const upcomingInterviews = computed(() => (workspace.value?.interviews || [])
  .filter((item) => item.status !== 'completed' && item.status !== 'cancelled')
  .slice(0, 3))
const pendingTasks = computed(() => tasks.value.filter((task) => task.status === 'pending'))
const assigneeDisplay = computed(() => {
  if (!newTask.value.assignee_user_id) return ''
  return selectedAssigneeName.value || `负责人 #${newTask.value.assignee_user_id}`
})

const latestActivityText = computed(() => {
  const value = workspace.value?.latest_activity_at || currentApplication.value?.applied_at || ''
  return value ? formatDateTime(value) : '-'
})

const candidateInitial = computed(() => {
  const name = workspace.value?.real_name?.trim() || '候'
  return name.slice(0, 1)
})

const candidateMeta = computed(() => {
  const items = [
    workspace.value?.education,
    workspace.value?.school,
    currentApplication.value?.job_title,
  ].filter(Boolean)
  return items.length ? items.join(' / ') : '暂无教育或岗位信息'
})

const loadWorkspace = async () => {
  if (!candidateUserId) return
  loading.value = true
  try {
    const data = await getCandidateWorkspace(candidateUserId)
    workspace.value = data.workspace
  } catch (err: any) {
    ElMessage.error('加载候选人信息失败：' + (err?.message || ''))
  } finally {
    loading.value = false
  }
}

const loadNotes = async () => {
  try {
    const data = await listNotes(candidateUserId)
    notes.value = data.list || []
  } catch {
    notes.value = []
  }
}

const loadAllTags = async () => {
  try {
    const data = await listTags()
    allTags.value = data.list || []
  } catch {
    allTags.value = []
  }
}

const loadTasks = async () => {
  try {
    const data = await listFollowUpTasks({ candidate_user_id: candidateUserId })
    tasks.value = data.list || []
  } catch {
    tasks.value = []
  }
}

const loadCollaborationData = async () => {
  await Promise.all([loadNotes(), loadAllTags(), loadTasks()])
  collaborationLoaded.value = true
}

const ensureSectionData = async (section: string) => {
  if (section === 'collaboration' && !collaborationLoaded.value) {
    await loadCollaborationData()
  }
}

const addNote = async () => {
  if (!newNoteContent.value.trim()) return
  noteLoading.value = true
  try {
    await createNote({
      candidate_user_id: candidateUserId,
      content: newNoteContent.value.trim(),
    })
    newNoteContent.value = ''
    ElMessage.success('备注已添加')
    await loadNotes()
  } catch (err: any) {
    ElMessage.error('添加备注失败：' + (err?.message || ''))
  } finally {
    noteLoading.value = false
  }
}

const addTagByName = async () => {
  const name = tagInput.value.trim()
  if (!name) {
    ElMessage.warning('请输入或选择标签')
    return
  }
  if (candidateTags.value.some((tag) => tag.name === name)) {
    ElMessage.warning('该标签已添加')
    tagInput.value = ''
    return
  }
  tagLoading.value = true
  try {
    const existing = allTags.value.find((tag) => tag.name === name)
    let tagId = Number(existing?.id || 0)
    if (!tagId) {
      const data = await createTag({ name })
      tagId = Number(data.tag?.id || 0)
      await loadAllTags()
    }
    if (!tagId) {
      ElMessage.error('标签创建失败，请稍后重试')
      return
    }
    await assignTag({ tag_id: tagId, candidate_user_id: candidateUserId })
    tagInput.value = ''
    ElMessage.success(existing ? '标签已添加' : '标签已创建并添加')
    await loadWorkspace()
  } catch (err: any) {
    ElMessage.error('添加标签失败：' + (err?.message || ''))
  } finally {
    tagLoading.value = false
  }
}

const removeTagFromCandidate = async (tagId: number) => {
  try {
    await unassignTag({ tag_id: tagId, candidate_user_id: candidateUserId })
    ElMessage.success('标签已移除')
    await loadWorkspace()
  } catch (err: any) {
    ElMessage.error('移除标签失败：' + (err?.message || ''))
  }
}

const openTaskDialog = () => {
  newTask.value = { title: '', description: '', assignee_user_id: 0, due_at: '' }
  selectedAssigneeName.value = ''
  taskDialogVisible.value = true
}

const handleAssigneeSelect = (user: StaffUserInfo) => {
  newTask.value.assignee_user_id = Number(user.user_id)
  selectedAssigneeName.value = user.username
}

const openResume = () => {
  const url = workspace.value?.resume_url?.trim()
  if (!url) {
    ElMessage.warning('简历链接暂不可用')
    return
  }
  window.open(url, '_blank', 'noopener')
}

const submitTask = async () => {
  if (!newTask.value.title.trim()) {
    ElMessage.warning('请输入任务标题')
    return
  }
  if (!newTask.value.assignee_user_id) {
    ElMessage.warning('请选择负责人')
    return
  }
  taskLoading.value = true
  try {
    await createFollowUpTask({
      candidate_user_id: candidateUserId,
      title: newTask.value.title.trim(),
      description: newTask.value.description,
      assignee_user_id: newTask.value.assignee_user_id,
      due_at: newTask.value.due_at || undefined,
    })
    ElMessage.success('任务已创建')
    taskDialogVisible.value = false
    await loadTasks()
  } catch (err: any) {
    ElMessage.error('创建任务失败：' + (err?.message || ''))
  } finally {
    taskLoading.value = false
  }
}

const completeTask = async (taskId: number) => {
  try {
    await completeFollowUpTask(taskId)
    ElMessage.success('任务已完成')
    await loadTasks()
  } catch (err: any) {
    ElMessage.error('完成任务失败：' + (err?.message || ''))
  }
}

const navigateSection = (section: string | number) => {
  router.push(`/hr/candidates/${candidateUserId}/${String(section)}`)
}

const goBackToLedger = () => {
  const applications = workspace.value?.applications || []
  const target = applications.find((app) => Number(app.is_current) === 1) || applications[0]
  const jobId = Number(target?.job_id || 0)
  if (jobId) {
    router.push(`/hr/jobs/${jobId}/applications`)
    return
  }
  router.back()
}

onMounted(async () => {
  await Promise.all([loadWorkspace(), loadTasks()])
  await ensureSectionData(activeSection.value)
})

watch(activeSection, (section) => {
  ensureSectionData(section)
})
</script>

<template>
  <div class="candidate-detail">
    <div v-loading="loading" class="candidate-detail__body">
      <template v-if="workspace">
        <section class="candidate-hero">
          <div class="candidate-hero__topline">
            <el-button class="back-button" text :icon="ArrowLeft" @click="goBackToLedger">返回</el-button>
            <div class="candidate-hero__actions">
              <el-button :icon="Document" :disabled="!workspace.resume_url" @click="openResume">查看简历</el-button>
              <el-button type="primary" :icon="Plus" @click="navigateSection('collaboration')">新增跟进</el-button>
            </div>
          </div>

          <div class="candidate-hero__main">
            <div class="candidate-avatar" aria-hidden="true">{{ candidateInitial }}</div>
            <div class="candidate-identity">
              <div class="candidate-identity__row">
                <h2>{{ workspace.real_name || '候选人详情' }}</h2>
                <el-tag v-if="currentApplication" :type="getStatusType(currentApplication.status_key)" size="large" effect="light">
                  {{ getHRStatusLabel(currentApplication.status_key) }}
                </el-tag>
              </div>
              <p>{{ candidateMeta }}</p>
              <div class="candidate-contact">
                <span v-if="workspace.phone">{{ workspace.phone }}</span>
                <span>最近动态：{{ latestActivityText }}</span>
              </div>
            </div>
          </div>

          <div class="candidate-kpis">
            <button class="kpi-tile" type="button" @click="navigateSection('applications')">
              <el-icon><Briefcase /></el-icon>
              <span>投递</span>
              <strong>{{ overviewStats.applications }}</strong>
            </button>
            <button class="kpi-tile" type="button" @click="navigateSection('process')">
              <el-icon><Clock /></el-icon>
              <span>面试</span>
              <strong>{{ overviewStats.interviews }}</strong>
            </button>
            <button class="kpi-tile" type="button" @click="navigateSection('process')">
              <el-icon><TrendCharts /></el-icon>
              <span>Offer</span>
              <strong>{{ overviewStats.offers }}</strong>
            </button>
            <button class="kpi-tile" type="button" @click="navigateSection('collaboration')">
              <el-icon><User /></el-icon>
              <span>待办</span>
              <strong>{{ pendingTasks.length }}</strong>
            </button>
          </div>
        </section>

        <el-tabs class="candidate-tabs" :model-value="activeSection" @tab-change="navigateSection">
          <el-tab-pane v-for="section in sectionOptions" :key="section.key" :label="section.label" :name="section.key" />
        </el-tabs>

        <section v-if="activeSection === 'overview'" class="candidate-section">
          <div class="overview-grid">
            <el-card shadow="never" class="detail-card resume-card">
              <template #header><span class="card-title">候选人档案</span></template>
              <div class="profile-grid">
                <div class="profile-field">
                  <span>姓名</span>
                  <strong>{{ workspace.real_name || '-' }}</strong>
                </div>
                <div class="profile-field">
                  <span>电话</span>
                  <strong>{{ workspace.phone || '-' }}</strong>
                </div>
                <div class="profile-field">
                  <span>学历</span>
                  <strong>{{ workspace.education || '-' }}</strong>
                </div>
                <div class="profile-field">
                  <span>毕业院校</span>
                  <strong>{{ workspace.school || '-' }}</strong>
                </div>
              </div>
              <div class="section-block">
                <div class="section-block__label">技能关键词</div>
                <div class="skill-cloud">
                  <el-tag v-for="skill in workspace.skills" :key="skill" effect="plain">{{ skill }}</el-tag>
                  <span v-if="!workspace.skills || workspace.skills.length === 0" class="no-data no-data--compact">暂无技能</span>
                </div>
              </div>
              <div class="section-block">
                <div class="section-block__label">工作经验</div>
                <div class="work-experience rich-content" v-html="workExperienceHtml"></div>
              </div>
            </el-card>

            <aside class="overview-side">
              <el-card shadow="never" class="detail-card">
                <template #header><span class="card-title">当前投递</span></template>
                <div v-if="currentApplication" class="current-flow">
                  <router-link :to="`/hr/jobs/${currentApplication.job_id}/applications`" class="current-flow__title">
                    {{ currentApplication.job_title }}
                  </router-link>
                  <el-tag :type="getStatusType(currentApplication.status_key)" effect="light">
                    {{ getHRStatusLabel(currentApplication.status_key) }}
                  </el-tag>
                  <div class="current-flow__meta">
                    <span>投递：{{ formatDateTime(currentApplication.applied_at) }}</span>
                    <span>第 {{ currentApplication.round_no || 1 }} 轮</span>
                    <span v-if="currentApplication.department">{{ currentApplication.department }}</span>
                    <span v-if="currentApplication.location">{{ currentApplication.location }}</span>
                  </div>
                </div>
                <div v-else class="no-data">暂无当前投递</div>
              </el-card>

              <el-card shadow="never" class="detail-card">
                <template #header><span class="card-title">标签</span></template>
                <div class="assigned-tags">
                  <el-tag v-for="tag in candidateTags" :key="tag.id">
                    {{ tag.name }}
                  </el-tag>
                  <span v-if="candidateTags.length === 0" class="no-data no-data--compact">暂无标签</span>
                </div>
              </el-card>

              <el-card shadow="never" class="detail-card">
                <template #header><span class="card-title">最近投递</span></template>
                <div class="compact-list">
                  <div v-for="app in recentApplications" :key="app.application_id" class="compact-row">
                    <span>{{ app.job_title }}</span>
                    <el-tag size="small" :type="getStatusType(app.status_key)">{{ getHRStatusLabel(app.status_key) }}</el-tag>
                  </div>
                  <div v-if="recentApplications.length === 0" class="no-data no-data--compact">暂无投递记录</div>
                </div>
              </el-card>

              <el-card shadow="never" class="detail-card">
                <template #header><span class="card-title">待面试</span></template>
                <div class="compact-list">
                  <div v-for="iv in upcomingInterviews" :key="iv.interview_id" class="compact-row">
                    <span>{{ iv.job_title || iv.title }}</span>
                    <em>{{ iv.scheduled_at ? formatDateTime(iv.scheduled_at) : '待定' }}</em>
                  </div>
                  <div v-if="upcomingInterviews.length === 0" class="no-data no-data--compact">暂无待面试</div>
                </div>
              </el-card>
            </aside>
          </div>
        </section>

        <section v-else-if="activeSection === 'applications'" class="candidate-section">
          <el-card shadow="never" class="detail-card">
            <template #header><span class="card-title">投递记录</span></template>
            <div class="application-list">
              <div v-for="app in workspace.applications" :key="app.application_id" class="application-item">
                <div class="app-header">
                  <router-link :to="`/hr/jobs/${app.job_id}/applications`" class="app-job-title">{{ app.job_title }}</router-link>
                  <div class="app-badges">
                    <el-tag :type="getStatusType(app.status_key)" size="small">{{ getHRStatusLabel(app.status_key) }}</el-tag>
                    <el-tag v-if="app.is_current" size="small" type="success">当前流程</el-tag>
                    <el-tag v-else size="small" type="info">历史</el-tag>
                  </div>
                </div>
                <div class="app-meta">
                  <span>投递时间：{{ formatDateTime(app.applied_at) }}</span>
                  <span>第 {{ app.round_no || 1 }} 轮</span>
                  <span v-if="app.department">{{ app.department }}</span>
                  <span v-if="app.location">{{ app.location }}</span>
                </div>
              </div>
              <div v-if="!workspace.applications || workspace.applications.length === 0" class="no-data">暂无投递记录</div>
            </div>
          </el-card>
        </section>

        <section v-else-if="activeSection === 'process'" class="candidate-section">
          <div class="process-grid">
            <div>
              <el-card shadow="never" class="detail-card">
                <template #header><span class="card-title">面试记录</span></template>
                <div class="interview-list">
                  <div v-for="iv in workspace.interviews" :key="iv.interview_id" class="detail-item">
                    <div class="detail-item-header">
                      <el-tag size="small" :type="iv.status === 'completed' ? 'success' : iv.status === 'cancelled' ? 'info' : 'warning'">
                        {{ iv.status === 'completed' ? '已完成' : iv.status === 'cancelled' ? '已取消' : '待面试' }}
                      </el-tag>
                      <span class="detail-item-round">第{{ iv.round_no }}轮</span>
                    </div>
                    <div class="detail-item-meta">
                      <span v-if="iv.scheduled_at">{{ formatDateTime(iv.scheduled_at) }}</span>
                      <span v-if="iv.interviewer_name">面试官：{{ iv.interviewer_name }}</span>
                      <span v-if="iv.job_title">岗位：{{ iv.job_title }}</span>
                    </div>
                  </div>
                  <div v-if="!workspace.interviews || workspace.interviews.length === 0" class="no-data">暂无面试记录</div>
                </div>
              </el-card>
            </div>
            <div>
              <el-card shadow="never" class="detail-card">
                <template #header><span class="card-title">Offer 记录</span></template>
                <div class="offer-list">
                  <div v-for="o in workspace.offers" :key="o.offer_id" class="detail-item">
                    <div class="detail-item-header">
                      <el-tag size="small" :type="o.status === 'accepted' ? 'success' : o.status === 'rejected' ? 'danger' : 'warning'">
                        {{ o.status === 'draft' ? '草稿' : o.status === 'sent' ? '已发送' : o.status === 'accepted' ? '已接受' : o.status === 'rejected' ? '已拒绝' : o.status === 'withdrawn' ? '已撤回' : o.status }}
                      </el-tag>
                      <span class="detail-item-round">{{ o.title }}</span>
                    </div>
                    <div class="detail-item-meta">
                      <span v-if="o.salary_range">薪资：{{ o.salary_range }}</span>
                      <span v-if="o.level">职级：{{ o.level }}</span>
                      <span v-if="o.work_location">地点：{{ o.work_location }}</span>
                      <span v-if="o.job_title">岗位：{{ o.job_title }}</span>
                    </div>
                  </div>
                  <div v-if="!workspace.offers || workspace.offers.length === 0" class="no-data">暂无 Offer 记录</div>
                </div>
              </el-card>
            </div>
          </div>
        </section>

        <section v-else-if="activeSection === 'collaboration'" class="candidate-section">
          <div class="collaboration-grid">
            <div class="collaboration-side">
              <el-card shadow="never" class="detail-card">
                <template #header><span class="card-title">标签</span></template>
                <div class="tags-section">
                  <div class="assigned-tags">
                    <el-tag
                      v-for="tag in candidateTags"
                      :key="tag.id"
                      closable
                      @close="removeTagFromCandidate(tag.id)"
                    >
                      {{ tag.name }}
                    </el-tag>
                    <span v-if="candidateTags.length === 0" class="no-data no-data--compact">暂无标签</span>
                  </div>
                  <el-divider />
                  <div class="tag-control">
                    <el-select
                      v-model="tagInput"
                      filterable
                      allow-create
                      default-first-option
                      clearable
                      placeholder="输入或选择标签"
                      size="small"
                      @keyup.enter="addTagByName"
                    >
                      <el-option v-for="tag in availableTags" :key="tag.id" :label="tag.name" :value="tag.name">
                        <span>{{ tag.name }}</span>
                      </el-option>
                    </el-select>
                    <el-button size="small" type="primary" :loading="tagLoading" :disabled="!tagInput.trim()" @click="addTagByName">
                      添加
                    </el-button>
                    <div v-if="availableTags.length === 0 && allTags.length > 0" class="tag-control__hint">可用标签都已添加</div>
                  </div>
                </div>
              </el-card>

              <el-card shadow="never" class="detail-card">
                <template #header>
                  <span class="card-title" style="display: flex; justify-content: space-between; align-items: center;">
                    <span>跟进任务</span>
                    <el-button size="small" type="primary" :icon="Plus" @click="openTaskDialog">新建任务</el-button>
                  </span>
                </template>
                <div class="task-list">
                  <div v-for="task in tasks" :key="task.id" class="task-item" :class="{ completed: task.status === 'completed' }">
                    <div class="task-header">
                      <span class="task-title">{{ task.title }}</span>
                      <el-tag v-if="task.status === 'pending'" size="small" type="warning">待完成</el-tag>
                      <el-tag v-else size="small" type="success">已完成</el-tag>
                    </div>
                    <div v-if="task.description" class="task-desc">{{ task.description }}</div>
                    <div class="task-meta">
                      <span>负责人：{{ task.assignee_name }}</span>
                      <span v-if="task.due_at">截止：{{ formatDateTime(task.due_at) }}</span>
                      <span v-if="task.completed_at">完成于：{{ formatDateTime(task.completed_at) }}</span>
                    </div>
                    <div v-if="task.status === 'pending'" class="task-actions">
                      <el-button size="small" type="success" plain @click="completeTask(task.id)">标记完成</el-button>
                    </div>
                  </div>
                  <div v-if="tasks.length === 0" class="no-data">暂无任务</div>
                </div>
              </el-card>
            </div>
            <div>
              <el-card shadow="never" class="detail-card">
                <template #header><span class="card-title">内部备注</span></template>
                <div class="notes-section">
                  <div class="note-input">
                    <el-input v-model="newNoteContent" type="textarea" :rows="3" placeholder="输入内部备注..." maxlength="2000" show-word-limit />
                    <el-button type="primary" size="small" style="margin-top: 8px;" :loading="noteLoading" :disabled="!newNoteContent.trim()" @click="addNote">
                      添加备注
                    </el-button>
                  </div>
                  <el-divider />
                  <div class="note-list">
                    <div v-for="note in notes" :key="note.id" class="note-item">
                      <div class="note-header">
                        <span class="note-author">{{ note.author_name }}</span>
                        <span class="note-time">{{ formatDateTime(note.created_at) }}</span>
                      </div>
                      <div class="note-content">{{ note.content }}</div>
                    </div>
                    <div v-if="notes.length === 0" class="no-data">暂无备注</div>
                  </div>
                </div>
              </el-card>
            </div>
          </div>
        </section>

        <section v-else-if="activeSection === 'timeline'" class="candidate-section">
          <el-card shadow="never" class="detail-card timeline-card">
            <template #header><span class="card-title">动态时间线</span></template>
            <TimelineView :candidate-user-id="candidateUserId" />
          </el-card>
        </section>
      </template>
    </div>

    <el-dialog v-model="taskDialogVisible" title="新建跟进任务" width="500px">
      <el-form :model="newTask" label-width="80px">
        <el-form-item label="任务标题" required>
          <el-input v-model="newTask.title" placeholder="请输入任务标题" />
        </el-form-item>
        <el-form-item label="任务描述">
          <el-input v-model="newTask.description" type="textarea" :rows="3" placeholder="请输入任务描述" />
        </el-form-item>
        <el-form-item label="负责人" required>
          <el-input
            :model-value="assigneeDisplay"
            readonly
            placeholder="请选择负责人"
            @click="assigneePickerVisible = true"
          >
            <template #append>
              <el-button @click.stop="assigneePickerVisible = true">选择</el-button>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item label="截止时间">
          <el-date-picker
            v-model="newTask.due_at"
            type="datetime"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
            placeholder="请选择截止时间"
            style="width: 100%"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="taskDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="taskLoading" @click="submitTask">确定</el-button>
      </template>
    </el-dialog>

    <InterviewerPickerDialog
      v-model:visible="assigneePickerVisible"
      title="选择负责人"
      empty-text="暂无可选负责人"
      select-label="选择"
      :selected-id="newTask.assignee_user_id"
      @select="handleAssigneeSelect"
    />

    <div v-if="!workspace && !loading" class="empty-state">
      <el-empty description="未找到候选人信息" />
    </div>
  </div>
</template>

<style scoped>
.candidate-detail {
  flex: 1;
  min-height: 0;
  width: 100%;
  box-sizing: border-box;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 18px 20px 28px;
  max-width: 1440px;
  margin: 0 auto;
  color: #1f2937;
}

.candidate-detail__body {
  min-height: 220px;
}

.candidate-hero {
  margin-bottom: 14px;
  padding: 18px;
  border: 1px solid #dbe4ef;
  border-radius: 8px;
  background:
    linear-gradient(135deg, rgba(239, 246, 255, 0.82), rgba(248, 250, 252, 0.94) 42%, rgba(240, 253, 244, 0.72)),
    #fff;
  box-shadow: 0 8px 28px rgba(15, 23, 42, 0.07);
}

.candidate-hero__topline,
.candidate-hero__actions,
.candidate-hero__main,
.candidate-identity__row,
.candidate-contact,
.candidate-kpis,
.assigned-tags,
.add-tag-row,
.app-badges {
  display: flex;
  align-items: center;
}

.candidate-hero__topline {
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 18px;
}

.candidate-hero__actions {
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.back-button {
  flex: 0 0 auto;
  padding-left: 0;
  color: #475569;
  font-weight: 600;
}

.back-button:hover {
  color: #1d4ed8;
  background: transparent;
}

.candidate-hero__main {
  min-width: 0;
  gap: 16px;
}

.candidate-avatar {
  flex: 0 0 64px;
  width: 64px;
  height: 64px;
  border-radius: 8px;
  display: grid;
  place-items: center;
  background: #1d4ed8;
  color: #fff;
  font-size: 28px;
  font-weight: 800;
  box-shadow: inset 0 -10px 18px rgba(15, 23, 42, 0.2);
}

.candidate-identity {
  min-width: 0;
  flex: 1;
}

.candidate-identity__row {
  min-width: 0;
  flex-wrap: wrap;
  gap: 10px;
}

.candidate-identity h2 {
  margin: 0;
  font-size: 26px;
  line-height: 1.2;
  font-weight: 800;
  color: #111827;
  overflow-wrap: anywhere;
}

.candidate-identity p {
  margin: 7px 0 0;
  color: #475569;
  font-size: 14px;
  line-height: 1.55;
  overflow-wrap: anywhere;
}

.candidate-contact {
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
  color: #64748b;
  font-size: 13px;
}

.candidate-contact span {
  padding: 4px 8px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid rgba(203, 213, 225, 0.72);
}

.candidate-kpis {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-top: 16px;
}

.kpi-tile {
  min-width: 0;
  min-height: 76px;
  padding: 12px;
  border: 1px solid rgba(203, 213, 225, 0.78);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.78);
  color: #475569;
  display: grid;
  grid-template-columns: 32px 1fr;
  grid-template-rows: auto auto;
  gap: 4px 10px;
  align-items: center;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.16s ease, box-shadow 0.16s ease, transform 0.16s ease;
}

.kpi-tile:hover {
  border-color: rgba(37, 99, 235, 0.42);
  box-shadow: 0 8px 20px rgba(15, 23, 42, 0.08);
  transform: translateY(-1px);
}

.kpi-tile .el-icon {
  grid-row: 1 / 3;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: grid;
  place-items: center;
  background: #eff6ff;
  color: #2563eb;
  font-size: 17px;
}

.kpi-tile span {
  min-width: 0;
  font-size: 12px;
  font-weight: 700;
}

.kpi-tile strong {
  min-width: 0;
  color: #0f172a;
  font-size: 24px;
  line-height: 1;
}

.candidate-tabs {
  margin-bottom: 18px;
  padding: 0 16px;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  box-shadow: none;
}

.candidate-tabs :deep(.el-tabs__header) {
  margin: 0;
}

.candidate-tabs :deep(.el-tabs__nav-wrap),
.candidate-tabs :deep(.el-tabs__nav-scroll),
.candidate-tabs :deep(.el-tabs__nav) {
  height: 48px;
}

.candidate-tabs :deep(.el-tabs__nav-scroll) {
  overflow-x: auto;
  scrollbar-width: none;
}

.candidate-tabs :deep(.el-tabs__nav-scroll::-webkit-scrollbar) {
  display: none;
}

.candidate-tabs :deep(.el-tabs__nav) {
  display: flex;
  gap: 20px;
  border: 0;
}

.candidate-tabs :deep(.el-tabs__nav-wrap::after) {
  display: none;
}

.candidate-tabs :deep(.el-tabs__active-bar) {
  height: 2px;
  background: #2563eb;
}

.candidate-tabs :deep(.el-tabs__item) {
  height: 48px;
  min-width: auto;
  padding: 0;
  box-sizing: border-box;
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  border-radius: 0;
  color: #64748b;
  font-size: 14px;
  font-weight: 600;
  line-height: 1;
  transition:
    background-color 0.16s ease,
    color 0.16s ease;
}

.candidate-tabs :deep(.el-tabs__item:first-child),
.candidate-tabs :deep(.el-tabs__item:last-child) {
  padding: 0;
}

.candidate-tabs :deep(.el-tabs__item:hover) {
  color: #2563eb;
  background: transparent;
}

.candidate-tabs :deep(.el-tabs__item.is-active) {
  color: #1d4ed8;
  background: transparent;
}

.candidate-tabs :deep(.el-tabs__item.is-focus) {
  box-shadow: none;
}

.candidate-section {
  min-height: 0;
}

.overview-grid,
.process-grid,
.collaboration-grid {
  display: grid;
  gap: 16px;
  align-items: start;
}

.overview-grid {
  grid-template-columns: minmax(0, 1fr) minmax(300px, 380px);
}

.process-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.collaboration-grid {
  grid-template-columns: minmax(300px, 420px) minmax(0, 1fr);
}

.collaboration-side {
  min-width: 0;
}

.detail-card {
  margin-bottom: 16px;
  border-radius: 8px;
  border-color: #e2e8f0;
}

.detail-card :deep(.el-card__header) {
  padding: 13px 16px;
  background: #f8fafc;
  border-bottom-color: #e2e8f0;
}

.detail-card :deep(.el-card__body) {
  padding: 16px;
}

.card-title {
  font-weight: 600;
  font-size: 16px;
  color: #1f2937;
}

.profile-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.profile-field {
  min-width: 0;
  padding: 12px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #fff;
}

.profile-field span,
.section-block__label {
  display: block;
  margin-bottom: 6px;
  color: #64748b;
  font-size: 12px;
  font-weight: 700;
}

.profile-field strong {
  display: block;
  color: #111827;
  font-size: 15px;
  line-height: 1.45;
  overflow-wrap: anywhere;
}

.section-block {
  margin-top: 16px;
}

.skill-cloud {
  min-height: 32px;
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
}

.work-experience {
  max-height: 320px;
  overflow-y: auto;
  overflow-x: hidden;
  color: #475569;
  line-height: 1.75;
  word-break: break-word;
}

.work-experience :deep(p),
.work-experience :deep(div) {
  margin: 0 0 8px;
}

.work-experience :deep(p:last-child),
.work-experience :deep(div:last-child) {
  margin-bottom: 0;
}

.work-experience :deep(ul),
.work-experience :deep(ol) {
  margin: 8px 0 12px 22px;
  padding: 0;
}

.work-experience :deep(li) {
  margin: 4px 0;
}

.work-experience :deep(strong),
.work-experience :deep(b) {
  color: #1e293b;
}

.current-flow {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.current-flow__title {
  color: #1d4ed8;
  font-size: 17px;
  font-weight: 700;
  line-height: 1.4;
  text-decoration: none;
  overflow-wrap: anywhere;
}

.current-flow__title:hover,
.app-job-title:hover {
  text-decoration: underline;
}

.current-flow__meta,
.compact-list,
.detail-item-meta,
.task-meta,
.app-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.current-flow__meta span,
.app-meta span,
.detail-item-meta span,
.task-meta span {
  color: #64748b;
  font-size: 12px;
  line-height: 1.6;
}

.compact-list {
  flex-direction: column;
}

.compact-row {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 9px 0;
  border-bottom: 1px solid #eef2f7;
}

.compact-row:last-child {
  border-bottom: 0;
}

.compact-row span:first-child {
  min-width: 0;
  color: #334155;
  font-weight: 600;
  overflow-wrap: anywhere;
}

.compact-row em {
  flex: 0 0 auto;
  color: #64748b;
  font-size: 12px;
  font-style: normal;
}

.no-data {
  color: #94a3b8;
  text-align: center;
  padding: 22px;
}

.no-data--compact {
  padding: 4px 0;
  text-align: left;
}

.assigned-tags,
.add-tag-row,
.app-badges {
  flex-wrap: wrap;
  gap: 7px;
}

.tag-control {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
}

.tag-control__hint {
  grid-column: 1 / -1;
  color: #94a3b8;
  font-size: 12px;
}

.application-item,
.detail-item,
.task-item,
.note-item {
  padding: 12px 0;
  border-bottom: 1px solid #eef2f7;
}

.application-item:last-child,
.detail-item:last-child,
.task-item:last-child,
.note-item:last-child {
  border-bottom: none;
}

.app-header,
.task-header,
.note-header,
.detail-item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 6px;
}

.app-job-title {
  font-weight: 600;
  color: #2563eb;
  text-decoration: none;
}

.detail-item-round,
.task-title,
.note-author {
  font-weight: 600;
  color: #1f2937;
}

.task-item.completed {
  opacity: 0.62;
}

.task-desc,
.note-content {
  margin-bottom: 6px;
  color: #475569;
  font-size: 13px;
  line-height: 1.65;
  white-space: pre-wrap;
  word-break: break-word;
}

.task-actions {
  margin-top: 6px;
}

.note-input {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
}

.note-time {
  color: #94a3b8;
  font-size: 12px;
}

.empty-state {
  padding: 60px 0;
}

@media (max-width: 1080px) {
  .candidate-hero__main {
    align-items: flex-start;
    flex-wrap: wrap;
  }

  .overview-grid,
  .process-grid,
  .collaboration-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 760px) {
  .candidate-detail {
    padding: 12px;
  }

  .candidate-hero {
    padding: 14px;
  }

  .candidate-hero__topline {
    align-items: flex-start;
    flex-direction: column;
  }

  .candidate-hero__actions {
    width: 100%;
    justify-content: flex-start;
  }

  .candidate-avatar {
    width: 54px;
    height: 54px;
    flex-basis: 54px;
    font-size: 23px;
  }

  .candidate-identity h2 {
    font-size: 22px;
  }

  .candidate-kpis,
  .profile-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .app-header,
  .task-header,
  .note-header,
  .detail-item-header,
  .compact-row {
    align-items: flex-start;
    flex-direction: column;
  }

  .candidate-tabs {
    margin-left: -2px;
    margin-right: -2px;
  }

  .candidate-tabs :deep(.el-tabs__item) {
    min-width: 86px;
    padding: 0 12px;
  }

  .candidate-tabs :deep(.el-tabs__item:first-child),
  .candidate-tabs :deep(.el-tabs__item:last-child) {
    padding: 0 12px;
  }
}

@media (max-width: 520px) {
  .candidate-kpis,
  .profile-grid {
    grid-template-columns: 1fr;
  }

  .candidate-hero__actions .el-button {
    margin-left: 0;
  }

  .tag-control {
    grid-template-columns: 1fr;
  }
}
</style>
