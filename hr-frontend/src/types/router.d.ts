import 'vue-router'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    requiresCandidate?: boolean
    requiresHR?: boolean
    requiresAdmin?: boolean
    title?: string
  }
}
