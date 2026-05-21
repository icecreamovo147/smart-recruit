<script setup lang="ts">
import { shallowRef, onBeforeUnmount } from 'vue'
import { Editor, Toolbar } from '@wangeditor/editor-for-vue'
import '@wangeditor/editor/dist/css/style.css'

interface EditorInstance {
  destroy(): void
}
const model = defineModel<string>({ default: '' })
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const editorRef = shallowRef<EditorInstance | null>(null)

const toolbarConfig = {
  toolbarKeys: [
    'bold',
    'italic',
    'underline',
    'color',
    'fontSize',
    'bulletedList',
    'numberedList',
    'justifyLeft',
    'justifyCenter',
    'justifyRight',
    'divider',
    'undo',
    'redo',
    'clearStyle',
  ],
}

const editorConfig = {
  placeholder: '请输入内容，可设置字号、颜色和列表格式',
  scroll: true,
  MENU_CONF: {
    fontSize: {
      fontSizeList: ['12px', '14px', '16px', '18px', '20px', '24px'],
    },
  },
}

const handleCreated = (editor: EditorInstance) => {
  editorRef.value = editor
}

onBeforeUnmount(() => {
  editorRef.value?.destroy()
})
</script>

<template>
  <div class="rich-editor">
    <Toolbar class="rich-editor__toolbar" :editor="editorRef" :default-config="toolbarConfig" mode="default" />
    <Editor v-model="model" class="rich-editor__body" :default-config="editorConfig" mode="default" @on-created="handleCreated" />
  </div>
</template>
