export interface NotificationItem {
  notification_id: number
  type: string
  title: string
  content: string
  link?: string
  biz_type?: string
  biz_id?: number
  is_read: boolean
  created_at: string
  read_at?: string
}

export interface NotificationListResponse {
  total: number
  list: NotificationItem[]
}

export interface NotificationSummaryResponse {
  unread: number
  latest_notification_id: number
  latest_created_at?: string
}

export interface UnreadCountResponse {
  unread: number
}

export interface NotificationStreamEvent {
  type: string
  notification_id: number
  unread: number
  title: string
  content: string
  link?: string
  created_at: string
  notification_type?: string
}
