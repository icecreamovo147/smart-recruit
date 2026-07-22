<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { ArrowLeft, ArrowRight, ZoomIn, ZoomOut } from '@element-plus/icons-vue'
import {
  AnnotationMode,
  getDocument,
  GlobalWorkerOptions,
  type PDFDocumentProxy,
  type RenderTask,
} from 'pdfjs-dist'
import pdfWorker from 'pdfjs-dist/build/pdf.worker.min.mjs?url'

GlobalWorkerOptions.workerSrc = pdfWorker

const props = defineProps<{
  url: string
}>()

const emit = defineEmits<{
  error: [message: string]
}>()

const MIN_SCALE = 0.5
const MAX_SCALE = 2.5
const SCALE_STEP = 0.1

const loading = ref(true)
const errorMessage = ref('')
const pageNumber = ref(1)
const pageCount = ref(0)
const scale = ref(1)
const canvasRef = ref<HTMLCanvasElement | null>(null)
const viewportRef = ref<HTMLElement | null>(null)

let pdfDoc: PDFDocumentProxy | null = null
let loadingTask: ReturnType<typeof getDocument> | null = null
let renderTask: RenderTask | null = null
let loadToken = 0
let renderToken = 0

const pageLabel = computed(() => {
  if (!pageCount.value) return '—'
  return `${pageNumber.value} / ${pageCount.value}`
})

const zoomLabel = computed(() => `${Math.round(scale.value * 100)}%`)
const canPrev = computed(() => pageNumber.value > 1)
const canNext = computed(() => pageNumber.value < pageCount.value)
const canZoomOut = computed(() => scale.value > MIN_SCALE + 0.001)
const canZoomIn = computed(() => scale.value < MAX_SCALE - 0.001)

const cancelRender = () => {
  if (!renderTask) return
  try {
    renderTask.cancel()
  } catch {
    // Ignore cancel races while switching pages/zoom.
  }
  renderTask = null
}

const destroyDocument = async () => {
  cancelRender()
  const task = loadingTask
  loadingTask = null
  const doc = pdfDoc
  pdfDoc = null
  if (task) {
    try {
      await task.destroy()
    } catch {
      // Ignore destroy races during remount/unmount.
    }
  }
  if (doc) {
    try {
      await doc.cleanup()
    } catch {
      // Ignore cleanup races during remount/unmount.
    }
  }
}

const renderPage = async () => {
  if (!pdfDoc || !canvasRef.value) return
  const token = ++renderToken
  cancelRender()

  const page = await pdfDoc.getPage(pageNumber.value)
  if (token !== renderToken) return

  const outputScale = window.devicePixelRatio || 1
  const viewport = page.getViewport({ scale: scale.value })
  const canvas = canvasRef.value
  const context = canvas.getContext('2d')
  if (!context) return

  canvas.width = Math.floor(viewport.width * outputScale)
  canvas.height = Math.floor(viewport.height * outputScale)
  canvas.style.width = `${Math.floor(viewport.width)}px`
  canvas.style.height = `${Math.floor(viewport.height)}px`

  const transform = outputScale !== 1 ? [outputScale, 0, 0, outputScale, 0, 0] : undefined
  renderTask = page.render({
    canvas,
    canvasContext: context,
    viewport,
    transform,
    annotationMode: AnnotationMode.DISABLE,
  })

  try {
    await renderTask.promise
  } catch (error: unknown) {
    const name = (error as { name?: string } | null)?.name
    if (name === 'RenderingCancelledException') return
    throw error
  } finally {
    if (token === renderToken) {
      renderTask = null
    }
  }
}

const fitWidth = async () => {
  if (!pdfDoc || !viewportRef.value) return
  const page = await pdfDoc.getPage(pageNumber.value)
  const base = page.getViewport({ scale: 1 })
  const available = Math.max(viewportRef.value.clientWidth - 32, 200)
  scale.value = Math.min(Math.max(available / base.width, MIN_SCALE), MAX_SCALE)
}

const loadDocument = async (url: string) => {
  const token = ++loadToken
  loading.value = true
  errorMessage.value = ''
  pageNumber.value = 1
  pageCount.value = 0
  await destroyDocument()

  try {
    loadingTask = getDocument({
      url,
      withCredentials: false,
    })
    const doc = await loadingTask.promise
    if (token !== loadToken) {
      await destroyDocument()
      return
    }
    pdfDoc = doc
    pageCount.value = doc.numPages
    await nextTick()
    await fitWidth()
    await renderPage()
  } catch {
    if (token !== loadToken) return
    errorMessage.value = '简历预览加载失败，请稍后重试'
    emit('error', errorMessage.value)
  } finally {
    if (token === loadToken) {
      loading.value = false
    }
  }
}

const goPrev = async () => {
  if (!canPrev.value) return
  pageNumber.value -= 1
  await renderPage()
}

const goNext = async () => {
  if (!canNext.value) return
  pageNumber.value += 1
  await renderPage()
}

const zoomBy = async (delta: number) => {
  const next = Math.min(MAX_SCALE, Math.max(MIN_SCALE, Number((scale.value + delta).toFixed(2))))
  if (Math.abs(next - scale.value) < 0.001) return
  scale.value = next
  await renderPage()
}

watch(
  () => props.url,
  (url) => {
    if (url) {
      void loadDocument(url)
    }
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  loadToken += 1
  renderToken += 1
  void destroyDocument()
})
</script>

<template>
  <div class="resume-pdf-preview" v-loading="loading">
    <div class="resume-pdf-toolbar">
      <div class="resume-pdf-toolbar__group">
        <el-button :disabled="!canPrev || loading" :icon="ArrowLeft" @click="goPrev">上一页</el-button>
        <span class="resume-pdf-toolbar__label">{{ pageLabel }}</span>
        <el-button :disabled="!canNext || loading" @click="goNext">
          下一页
          <el-icon class="el-icon--right"><ArrowRight /></el-icon>
        </el-button>
      </div>
      <div class="resume-pdf-toolbar__group">
        <el-button :disabled="!canZoomOut || loading" :icon="ZoomOut" @click="zoomBy(-SCALE_STEP)">缩小</el-button>
        <span class="resume-pdf-toolbar__label">{{ zoomLabel }}</span>
        <el-button :disabled="!canZoomIn || loading" :icon="ZoomIn" @click="zoomBy(SCALE_STEP)">放大</el-button>
      </div>
    </div>

    <div v-if="errorMessage" class="resume-preview-fallback">
      <h2>{{ errorMessage }}</h2>
      <p>若持续失败，可能是对象存储跨域未允许浏览器读取文件</p>
    </div>
    <div v-else ref="viewportRef" class="resume-pdf-viewport">
      <canvas ref="canvasRef" class="resume-pdf-canvas" />
    </div>
  </div>
</template>
