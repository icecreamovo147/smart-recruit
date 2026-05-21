<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { Ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getJobDetail } from '@/api/job'
import { applyJob } from '@/api/application'
import { useAuthStore } from '@/stores/auth'
import { renderRichText } from '@/utils/richText'
import { BusinessError } from '@/types/api'
import type { Job } from '@/types/domain'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const job = ref<Job | null>(null) as Ref<Job | null>
const loading = ref(false)
const applying = ref(false)
const applied = ref(false)

const load = async () => {
  loading.value = true
  try {
    job.value = await getJobDetail(Number(route.params.jobId))
  } finally {
    loading.value = false
  }
}

const apply = async () => {
  applying.value = true
  try {
    await applyJob({ job_id: Number(route.params.jobId) })
    applied.value = true
    ElMessage.success('投递成功')
  } catch (error: unknown) {
    const code = error instanceof BusinessError ? error.code : 0
    if (code === 4001) {
      ElMessageBox.confirm('请先完善个人资料后再投递', '资料未完善', { confirmButtonText: '去完善', cancelButtonText: '稍后' })
        .then(() => router.push('/profile'))
        .catch(() => {})
    }
    if (code === 4002) {
      ElMessageBox.confirm('请先上传简历后再投递', '简历未上传', { confirmButtonText: '去上传', cancelButtonText: '稍后' })
        .then(() => router.push('/resume'))
        .catch(() => {})
    }
    if (code === 4003 || code === 4004) {
      // 拦截器已提示，这里不再弹窗，也不改变按钮状态
      return
    }
  } finally {
    applying.value = false
  }
}

onMounted(load)
</script>

<template>
  <section>
    <el-skeleton v-if="loading" :rows="8" animated />
    <div v-else-if="job" class="detail-grid">
      <article class="content-surface">
        <div class="breadcrumb">首页 / 岗位列表 / {{ job.title }}</div>
        <h1 class="page-title">{{ job.title }}</h1>
        <p class="job-meta">{{ job.department }} · {{ job.location }} · {{ job.salary_range ? job.salary_range + ' 元/月' : '薪资面议' }}</p>
        <el-divider />
        <h3>岗位描述</h3>
        <div class="rich-content" v-html="renderRichText(job.description, '暂无岗位描述')"></div>
        <h3>任职要求</h3>
        <div class="rich-content" v-html="renderRichText(job.requirements, '暂无任职要求')"></div>
      </article>
      <aside class="content-surface detail-card">
        <div class="detail-card__body">
          <h3 class="card-title">投递岗位</h3>
          <p class="card-hint">完善资料与简历可提升投递成功率。</p>
        </div>
        <div class="detail-card__action">
          <el-button v-if="auth.isLoggedIn" type="primary" :loading="applying" :disabled="applied" style="width: 100%" @click="apply">
            {{ applied ? '已投递' : '立即投递' }}
          </el-button>
          <el-button v-else type="primary" style="width: 100%" @click="router.push('/login')">登录后可投递</el-button>
        </div>
      </aside>
    </div>
  </section>
</template>
