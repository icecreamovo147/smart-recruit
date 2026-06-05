<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft, Plus, Delete } from '@element-plus/icons-vue'
import { getCandidateWorkspace, createNote, listNotes, createTag, listTags, assignTag, unassignTag, listCandidateTags, createFollowUpTask, listFollowUpTasks, completeFollowUpTask } from '@/api/collaboration'
import { getHRStatusLabel, getStatusType } from '@/types/domain'
import type { CandidateWorkspace, CandidateNoteInfo, CandidateTagInfo, FollowUpTaskInfo } from '@/types/domain'

const route = useRoute()
const router = useRouter()

const candidateUserId = Number(route.params.candidateUserId)
const loading = ref(false)
const workspace = ref<CandidateWorkspace | null>(null)

// Notes
const notes = ref<CandidateNoteInfo[]>([])
const newNoteContent = ref('')
const noteLoading = ref(false)

// Tags (all available)
const allTags = ref<CandidateTagInfo[]>([])
const newTagName = ref('')
const newTagColor = ref('#409eff')
const tagLoading = ref(false)

// Follow-up tasks
const tasks = ref<FollowUpTaskInfo[]>([])
const taskLoading = ref(false)

// Dialog states
const taskDialogVisible = ref(false)
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

const candidateTags = computed<CandidateTagInfo[]>(() => workspace.value?.tags || [])

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

const createNewTag = async () => {
  if (!newTagName.value.trim()) return
  tagLoading.value = true
  try {
    await createTag({
      name: newTagName.value.trim(),
      color: newTagColor.value,
    })
    newTagName.value = ''
    ElMessage.success('标签已创建')
    await loadAllTags()
  } catch (err: any) {
    ElMessage.error('创建标签失败：' + (err?.message || ''))
  } finally {
    tagLoading.value = false
  }
}

const assignTagToCandidate = async (tagId: number) => {
  try {
    await assignTag({ tag_id: tagId, candidate_user_id: candidateUserId })
    ElMessage.success('标签已分配')
    await loadWorkspace()
  } catch (err: any) {
    ElMessage.error('分配标签失败：' + (err?.message || ''))
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
  taskDialogVisible.value = true
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

// Check if a tag is already assigned to the candidate
const isTagAssigned = (tagId: number): boolean => {
  return candidateTags.value.some((t: CandidateTagInfo) => t.id === tagId)
}

onMounted(async () => {
  await loadWorkspace()
  await Promise.all([
    loadNotes(),
    loadAllTags(),
    loadTasks(),
  ])
})
</script>

<template>
  <div class="candidate-detail">
    <!-- Header -->
    <div class="page-header">
      <el-button :icon="ArrowLeft" @click="router.back()">返回</el-button>
      <h2>候选人详情</h2>
    </div>

    <div v-loading="loading">
      <template v-if="workspace">
        <el-row :gutter="20">
          <!-- Left column: Profile, Tags, Notes -->
          <el-col :span="14">
            <!-- Profile Card -->
            <el-card shadow="never" class="detail-card">
              <template #header>
                <span class="card-title">基本信息</span>
              </template>
              <el-descriptions :column="2" border>
                <el-descriptions-item label="姓名">{{ workspace.real_name || '-' }}</el-descriptions-item>
                <el-descriptions-item label="电话">{{ workspace.phone || '-' }}</el-descriptions-item>
                <el-descriptions-item label="学历">{{ workspace.education || '-' }}</el-descriptions-item>
                <el-descriptions-item label="毕业院校">{{ workspace.school || '-' }}</el-descriptions-item>
                <el-descriptions-item label="技能" :span="2">
                  <el-tag v-for="skill in workspace.skills" :key="skill" size="small" style="margin-right: 4px; margin-bottom: 4px;">
                    {{ skill }}
                  </el-tag>
                  <span v-if="!workspace.skills || workspace.skills.length === 0">-</span>
                </el-descriptions-item>
                <el-descriptions-item label="工作经验" :span="2">
                  <div class="work-experience">{{ workspace.work_experience || '-' }}</div>
                </el-descriptions-item>
              </el-descriptions>
            </el-card>

            <!-- Tags Card -->
            <el-card shadow="never" class="detail-card">
              <template #header>
                <span class="card-title">标签</span>
              </template>
              <div class="tags-section">
                <div class="assigned-tags">
                  <el-tag
                    v-for="tag in candidateTags"
                    :key="tag.id"
                    :color="tag.color"
                    :style="{ color: '#fff', marginRight: '6px', marginBottom: '6px', cursor: 'pointer' }"
                    closable
                    @close="removeTagFromCandidate(tag.id)"
                  >
                    {{ tag.name }}
                  </el-tag>
                  <span v-if="candidateTags.length === 0" class="no-data">暂无标签</span>
                </div>
                <el-divider />
                <div class="add-tag-row">
                  <el-select
                    v-model="newTagName"
                    filterable
                    allow-create
                    default-first-option
                    placeholder="选择或创建标签"
                    size="small"
                    style="width: 200px; margin-right: 8px;"
                    @change="(val: string) => { if (val) { newTagName = val } }"
                  >
                    <el-option
                      v-for="tag in allTags"
                      :key="tag.id"
                      :label="tag.name"
                      :value="tag.name"
                    >
                      <span :style="{ color: tag.color }">{{ tag.name }}</span>
                    </el-option>
                  </el-select>
                  <el-popover placement="bottom" width="240" trigger="click">
                    <template #reference>
                      <el-button size="small" type="primary" :icon="Plus">添加标签</el-button>
                    </template>
                    <div style="display: flex; gap: 8px; flex-direction: column;">
                      <el-input v-model="newTagName" placeholder="标签名称" size="small" />
                      <el-color-picker v-model="newTagColor" size="small" />
                      <el-button size="small" type="primary" :loading="tagLoading" @click="createNewTag">
                        创建标签
                      </el-button>
                      <el-button
                        v-if="!isTagAssigned(0)"
                        size="small"
                        type="success"
                        @click="assignTagToCandidate(allTags.find((t: CandidateTagInfo) => t.name === newTagName)?.id || 0)"
                      >
                        分配已有标签
                      </el-button>
                    </div>
                  </el-popover>
                </div>
              </div>
            </el-card>

            <!-- Notes Card -->
            <el-card shadow="never" class="detail-card">
              <template #header>
                <span class="card-title">内部备注</span>
              </template>
              <div class="notes-section">
                <div class="note-input">
                  <el-input
                    v-model="newNoteContent"
                    type="textarea"
                    :rows="3"
                    placeholder="输入内部备注..."
                    maxlength="2000"
                    show-word-limit
                  />
                  <el-button
                    type="primary"
                    size="small"
                    style="margin-top: 8px;"
                    :loading="noteLoading"
                    :disabled="!newNoteContent.trim()"
                    @click="addNote"
                  >
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
          </el-col>

          <!-- Right column: Applications, Tasks, Stats -->
          <el-col :span="10">
            <!-- Stats Card -->
            <el-card shadow="never" class="detail-card">
              <template #header>
                <span class="card-title">概览</span>
              </template>
              <el-row :gutter="12">
                <el-col :span="8">
                  <div class="stat-item">
                    <div class="stat-number">{{ workspace.total_applications }}</div>
                    <div class="stat-label">投递</div>
                  </div>
                </el-col>
                <el-col :span="8">
                  <div class="stat-item">
                    <div class="stat-number">{{ workspace.total_interviews }}</div>
                    <div class="stat-label">面试</div>
                  </div>
                </el-col>
                <el-col :span="8">
                  <div class="stat-item">
                    <div class="stat-number">{{ workspace.total_offers }}</div>
                    <div class="stat-label">Offer</div>
                  </div>
                </el-col>
              </el-row>
            </el-card>

            <!-- Applications Card -->
            <el-card shadow="never" class="detail-card">
              <template #header>
                <span class="card-title">投递记录</span>
              </template>
              <div class="application-list">
                <div v-for="app in workspace.applications" :key="app.application_id" class="application-item">
                  <div class="app-header">
                    <router-link :to="`/hr/jobs/${app.job_id}/applications`" class="app-job-title">
                      {{ app.job_title }}
                    </router-link>
                    <el-tag :type="getStatusType(app.status_key)" size="small">
                      {{ getHRStatusLabel(app.status_key) }}
                    </el-tag>
                  </div>
                  <div class="app-meta">
                    <span>投递时间: {{ formatDateTime(app.applied_at) }}</span>
                    <span v-if="app.round_no > 1"> | 第{{ app.round_no }}轮</span>
                    <span v-if="app.is_current"> | <el-tag size="small" type="success">当前流程</el-tag></span>
                    <span v-else> | <el-tag size="small" type="info">历史</el-tag></span>
                  </div>
                </div>
                <div v-if="!workspace.applications || workspace.applications.length === 0" class="no-data">暂无投递记录</div>
              </div>
            </el-card>

            <!-- Follow-up Tasks Card -->
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
                    <span>负责人: {{ task.assignee_name }}</span>
                    <span v-if="task.due_at"> | 截止: {{ formatDateTime(task.due_at) }}</span>
                    <span v-if="task.completed_at"> | 完成于: {{ formatDateTime(task.completed_at) }}</span>
                  </div>
                  <div v-if="task.status === 'pending'" class="task-actions">
                    <el-button size="small" type="success" plain @click="completeTask(task.id)">标记完成</el-button>
                  </div>
                </div>
                <div v-if="tasks.length === 0" class="no-data">暂无任务</div>
              </div>
            </el-card>
          </el-col>
        </el-row>
      </template>
    </div>

    <!-- Task Dialog -->
    <el-dialog v-model="taskDialogVisible" title="新建跟进任务" width="500px">
      <el-form :model="newTask" label-width="80px">
        <el-form-item label="任务标题" required>
          <el-input v-model="newTask.title" placeholder="请输入任务标题" />
        </el-form-item>
        <el-form-item label="任务描述">
          <el-input v-model="newTask.description" type="textarea" :rows="3" placeholder="请输入任务描述" />
        </el-form-item>
        <el-form-item label="负责人" required>
          <el-input-number v-model="newTask.assignee_user_id" :min="1" placeholder="负责人用户ID" />
        </el-form-item>
        <el-form-item label="截止时间">
          <el-input v-model="newTask.due_at" placeholder="RFC 3339 格式，如 2026-06-30T23:59:59Z" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="taskDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="taskLoading" @click="submitTask">确定</el-button>
      </template>
    </el-dialog>

    <!-- Loading overlay -->
    <div v-if="!workspace && !loading" class="empty-state">
      <el-empty description="未找到候选人信息" />
    </div>
  </div>
</template>

<style scoped>
.candidate-detail {
  padding: 20px;
  max-width: 1400px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
}

.detail-card {
  margin-bottom: 16px;
}

.card-title {
  font-weight: 600;
  font-size: 16px;
}

.work-experience {
  white-space: pre-wrap;
  max-height: 120px;
  overflow-y: auto;
}

.no-data {
  color: #999;
  text-align: center;
  padding: 20px;
}

.tags-section {
  min-height: 60px;
}

.assigned-tags {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
}

.add-tag-row {
  display: flex;
  align-items: center;
}

.note-input {
  margin-bottom: 8px;
}

.note-item {
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
}

.note-item:last-child {
  border-bottom: none;
}

.note-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.note-author {
  font-weight: 600;
  font-size: 13px;
  color: #409eff;
}

.note-time {
  font-size: 12px;
  color: #999;
}

.note-content {
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
}

.stat-item {
  text-align: center;
  padding: 12px 0;
}

.stat-number {
  font-size: 28px;
  font-weight: 700;
  color: #409eff;
}

.stat-label {
  font-size: 13px;
  color: #666;
  margin-top: 4px;
}

.application-item {
  padding: 10px 0;
  border-bottom: 1px solid #f0f0f0;
}

.application-item:last-child {
  border-bottom: none;
}

.app-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.app-job-title {
  font-weight: 600;
  color: #409eff;
  text-decoration: none;
}

.app-job-title:hover {
  text-decoration: underline;
}

.app-meta {
  font-size: 12px;
  color: #999;
}

.task-item {
  padding: 10px 0;
  border-bottom: 1px solid #f0f0f0;
}

.task-item:last-child {
  border-bottom: none;
}

.task-item.completed {
  opacity: 0.6;
}

.task-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.task-title {
  font-weight: 600;
}

.task-desc {
  font-size: 13px;
  color: #666;
  margin-bottom: 4px;
}

.task-meta {
  font-size: 12px;
  color: #999;
  margin-bottom: 4px;
}

.task-actions {
  margin-top: 4px;
}

.empty-state {
  padding: 60px 0;
}
</style>
