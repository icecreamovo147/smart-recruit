/// <reference types="vite/client" />

import 'vue-router'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    requiresPermission?: string
    title?: string
  }
}

declare module 'element-plus/dist/locale/zh-cn.mjs'
