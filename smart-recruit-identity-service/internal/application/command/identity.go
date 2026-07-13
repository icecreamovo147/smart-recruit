package command

type Register struct {
	Username   string
	Password   string
	Email      string
	Role       int32
	InviteCode string
}

type Login struct {
	Username  string
	Password  string
	ClientIP  string
	UserAgent string
}

type RefreshToken struct {
	RefreshToken string
	ClientIP     string
	UserAgent    string
}

type RevokeRefreshToken struct {
	RefreshToken string
}

type UpdateEmail struct {
	UserID int64
	Email  string
}

type RecordAuthDecision struct {
	ActorUserID   uint64
	ActorRoles    string
	PermissionKey string
	ResourceType  string
	ResourceID    uint64
	Decision      string
	Reason        string
	RequestID     string
	ClientIP      string
}

type AssignUserRole struct {
	UserID  int64
	AdminID int64
	RoleKey string
}

type RevokeUserRole struct {
	UserID  int64
	AdminID int64
	RoleKey string
}

type AssignDataScope struct {
	UserID       int64
	AdminID      int64
	ScopeKey     string
	ResourceType string
	ResourceID   uint64
}

type RevokeDataScope struct {
	ScopeID uint64
	AdminID int64
}

type CreateStaffUser struct {
	Username string
	Password string
	Email    string
	AdminID  int64
	RoleKeys []string
}
