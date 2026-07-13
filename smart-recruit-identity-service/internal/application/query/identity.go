package query

type GetPrincipal struct {
	UserID int64
}

type ListStaffUsers struct {
	Page     int32
	PageSize int32
	Status   string
}

type GetUserRoles struct {
	UserID int64
}

type QueryAuthAuditLogs struct {
	ActorUserID   *uint64
	PermissionKey string
	Decision      string
	Page          int32
	PageSize      int32
}
