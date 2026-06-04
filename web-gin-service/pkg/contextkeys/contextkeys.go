package contextkeys

type CtxKey string

const (
	RequestID   CtxKey = "request_id"
	ClientIP    CtxKey = "client_ip"
	UserID      CtxKey = "auth_user_id"
	AccountType CtxKey = "auth_account_type"
)
