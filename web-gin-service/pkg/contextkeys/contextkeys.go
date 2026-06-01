package contextkeys

type CtxKey string

const (
	RequestID CtxKey = "request_id"
	ClientIP  CtxKey = "client_ip"
)
