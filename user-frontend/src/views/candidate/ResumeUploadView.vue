<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { Document, Download, UploadFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { confirmResume, getResume, presignResume, putResumeFile } from '@/api/resume'
import type { ResumeInfo } from '@/types/domain'

const file = ref<File | null>(null)
const loading = ref(false)
const resumeLoading = ref(false)
const currentResume = ref<ResumeInfo | null>(null)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const uploadRef = ref<any>(null)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const uploaderSectionRef = ref<any>(null)
const isPDF = computed(() => currentResume.value?.file_type === 'pdf')
const showUploader = ref(false)
const highlightUploader = ref(false)
const allowed = ['pdf', 'docx']
let highlightTimer: ReturnType<typeof setTimeout> | null = null

const loadResume = async () => {
  resumeLoading.value = true
  try {
    const data = await getResume()
    currentResume.value = data.resume || null
    showUploader.value = !currentResume.value
  } finally {
    resumeLoading.value = false
  }
}

const pick = (uploadFile: { raw: File; name: string; size: number }) => {
  const raw = uploadFile.raw
  const ext = raw.name.split('.').pop()!.toLowerCase()
  if (!allowed.includes(ext)) {
    ElMessage.error('仅支持 PDF、DOCX 格式')
    uploadRef.value?.clearFiles()
    file.value = null
    return false
  }
  if (raw.size > 20 * 1024 * 1024) {
    ElMessage.error('文件大小不能超过 20MB')
    uploadRef.value?.clearFiles()
    file.value = null
    return false
  }
  file.value = raw
  return false
}

const upload = async () => {
  if (!file.value) {
    ElMessage.warning('请选择简历文件')
    return
  }
  loading.value = true
  try {
    const ext = file.value.name.split('.').pop()!.toLowerCase()
    const presign = await presignResume({ file_name: file.value.name, file_type: ext })
    await putResumeFile(presign.upload_url, file.value)
    await confirmResume({ oss_key: presign.oss_key, file_name: file.value.name, file_type: ext, file_size: file.value.size, upload_id: presign.upload_id })
    ElMessage.success('简历上传成功')
    file.value = null
    uploadRef.value?.clearFiles()
    await loadResume()
  } catch (error: unknown) {
    const err = error as { code?: number; message?: string }
    if (err.code === 42911 || err.code === 42912) {
      // Rate limit errors already shown by the request interceptor.
      // Prevent subsequent OSS PUT / confirm calls by not continuing.
      uploadRef.value?.clearFiles()
      return
    }
    if (typeof err.code === 'string' && err.code === 'ERR_NETWORK') {
      ElMessage.error('上传到 OSS 失败，请检查 COS 跨域 CORS 配置')
      return
    }
    // Avoid duplicate error toast — the request interceptor already shows it for BusinessErrors.
    if (!err.code || typeof err.code === 'string') {
      ElMessage.error(err.message || '简历上传失败')
    }
  } finally {
    loading.value = false
  }
}

const showUpdateUploader = async () => {
  showUploader.value = true
  highlightUploader.value = true
  await nextTick()
  uploaderSectionRef.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  if (highlightTimer) clearTimeout(highlightTimer)
  highlightTimer = setTimeout(() => {
    highlightUploader.value = false
  }, 1600)
}

const downloadResume = () => {
  if (currentResume.value?.resume_url) {
    window.open(currentResume.value.resume_url, '_blank', 'noopener')
  }
}

const cancelUpdate = () => {
  showUploader.value = false
  file.value = null
  uploadRef.value?.clearFiles()
}

const formatFileSize = (value: number): string => {
  if (!value) return '0 KB'
  if (value >= 1024 * 1024) return `${(value / 1024 / 1024).toFixed(1)} MB`
  return `${Math.ceil(value / 1024)} KB`
}

const formatUploadedAt = (value: string): string => {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { hour12: false })
}

onMounted(loadResume)
</script>

<template>
  <section>
    <div class="page-header">
      <div>
        <h1 class="page-title">简历上传</h1>
        <p class="page-subtitle">请上传最新版本的简历文件。</p>
      </div>
    </div>
    <div class="content-surface" v-loading="resumeLoading">
      <div v-if="currentResume" class="resume-current">
        <div class="resume-current__icon">
          <el-icon><Document /></el-icon>
        </div>
        <div class="resume-current__body">
          <div class="resume-current__top">
            <h2>{{ currentResume.file_name }}</h2>
            <el-tag type="success">已上传</el-tag>
          </div>
          <div class="resume-meta">
            <span>{{ (currentResume.file_type || 'pdf').toUpperCase() }}</span>
            <span>{{ formatFileSize(currentResume.file_size) }}</span>
            <span>{{ formatUploadedAt(currentResume.uploaded_at) }}</span>
          </div>
        </div>
        <el-button type="primary" :icon="UploadFilled" @click="showUpdateUploader">更新简历</el-button>
      </div>

      <div v-if="currentResume" class="resume-preview">
        <iframe v-if="currentResume.resume_url && isPDF" :key="currentResume.resume_url" :src="currentResume.resume_url" title="简历预览" />
        <div v-else-if="currentResume.resume_url && !isPDF" class="resume-preview-fallback">
          <el-icon :size="48"><Document /></el-icon>
          <h2>DOCX 文件不支持在线预览</h2>
          <p>请下载后使用 Microsoft Word 或 WPS 打开查看</p>
          <el-button type="primary" :icon="Download" class="resume-download-btn" @click="downloadResume">
            下载简历文件
          </el-button>
        </div>
        <div v-else class="resume-preview-empty">
          <el-icon><Document /></el-icon>
          <h2>简历预览暂不可用</h2>
        </div>
      </div>

      <div v-if="showUploader" ref="uploaderSectionRef" class="resume-uploader" :class="{ 'resume-uploader--highlight': highlightUploader }">
        <div class="section-head">
          <h2>{{ currentResume ? '更新简历' : '上传简历' }}</h2>
          <el-button v-if="currentResume" @click="cancelUpdate">取消</el-button>
        </div>
        <el-upload ref="uploadRef" drag accept=".pdf,.docx,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document" :auto-upload="false" :limit="1" :on-change="pick" :show-file-list="Boolean(file)">
          <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
          <div class="el-upload__text">拖拽文件到此处，或点击选择</div>
          <template #tip>
            <div class="el-upload__tip">支持 PDF、DOCX，最大 20MB</div>
          </template>
        </el-upload>
        <el-button type="primary" :loading="loading" style="margin-top: 16px" @click="upload">
          {{ currentResume ? '上传新版本' : '上传简历' }}
        </el-button>
      </div>
    </div>
  </section>
</template>
