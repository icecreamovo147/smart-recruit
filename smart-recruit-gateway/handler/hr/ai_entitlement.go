package hr

import (
	"github.com/gin-gonic/gin"

	base "smart-recruit-gateway/handler"
	"smart-recruit-gateway/middleware"
	"smart-recruit-gateway/rpc"
	"smart-recruit-proto/recruitment/pb"
)

// requireTenantAICapability blocks paid AI entry points before any expensive
// downstream work begins. It also guarantees that the plan is pinned to an
// immutable platform capability release rather than merely enabling a flag.
func requireTenantAICapability(c *gin.Context, clients *rpc.Clients, capability string) bool {
	return resolveTenantAICapabilityVersion(c, clients, capability) > 0
}

func resolveTenantAICapabilityVersion(c *gin.Context, clients *rpc.Clients, capability string) int64 {
	if clients == nil || clients.Billing == nil {
		base.Internal(c, nil)
		return 0
	}
	access, err := clients.Billing.CheckAIAccess(c.Request.Context(), &pb.CheckAIAccessRequest{
		Owner:      &pb.BillingOwner{Type: pb.BillingOwnerType_BILLING_OWNER_TYPE_TENANT, Id: middleware.TenantID(c)},
		UserId:     middleware.UserID(c),
		Capability: capability,
	})
	if err != nil {
		base.Internal(c, err)
		return 0
	}
	if access.GetCode() != 0 || !access.GetAllowed() {
		base.From(c, 403, access.GetReason(), nil)
		return 0
	}
	if access.GetCapabilityVersionId() <= 0 {
		base.From(c, 503, "AI capability release is unavailable", nil)
		return 0
	}
	return access.GetCapabilityVersionId()
}
