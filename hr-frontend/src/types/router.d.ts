import 'vue-router'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    requiresPermission?: string     // RBAC permission key required
    requiresAnyPermission?: string[] // Any of these RBAC permission keys
    requiresRole?: string           // RBAC role key required
    title?: string
  }
}
