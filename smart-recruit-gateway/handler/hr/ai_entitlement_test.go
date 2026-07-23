package hr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"smart-recruit-gateway/rpc"
	"smart-recruit-proto/recruitment/pb"
)

type entitlementBillingClient struct {
	pb.BillingServiceClient
	response *pb.CheckAIAccessResponse
	request  *pb.CheckAIAccessRequest
}

func (c *entitlementBillingClient) CheckAIAccess(_ context.Context, req *pb.CheckAIAccessRequest, _ ...grpc.CallOption) (*pb.CheckAIAccessResponse, error) {
	c.request = req
	return c.response, nil
}

func TestRequireTenantAICapabilityRequiresPinnedRelease(t *testing.T) {
	gin.SetMode(gin.TestMode)
	billing := &entitlementBillingClient{response: &pb.CheckAIAccessResponse{Code: 0, Allowed: true}}
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		c.Set("user_id", int64(7))
		c.Set("tenant_id", int64(9))
		if requireTenantAICapability(c, &rpc.Clients{Billing: billing}, "ai.resume_parse") {
			c.Status(http.StatusNoContent)
		}
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/test", nil))

	var body struct {
		Code int32 `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != 503 {
		t.Fatalf("code = %d, want 503; body=%s", body.Code, recorder.Body.String())
	}
	if billing.request.GetCapability() != "ai.resume_parse" || billing.request.GetOwner().GetId() != 9 || billing.request.GetUserId() != 7 {
		t.Fatalf("billing request = %#v", billing.request)
	}
}

func TestRequireTenantAICapabilityAllowsPublishedRelease(t *testing.T) {
	gin.SetMode(gin.TestMode)
	billing := &entitlementBillingClient{response: &pb.CheckAIAccessResponse{Code: 0, Allowed: true, CapabilityVersionId: 81}}
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		c.Set("user_id", int64(7))
		c.Set("tenant_id", int64(9))
		if requireTenantAICapability(c, &rpc.Clients{Billing: billing}, "ai.match_evaluation") {
			c.Status(http.StatusNoContent)
		}
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/test", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
}
