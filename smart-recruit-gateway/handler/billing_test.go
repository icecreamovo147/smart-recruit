package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"smart-recruit-gateway/rpc"
	"smart-recruit-proto/recruitment/pb"
)

type webhookBillingClient struct {
	pb.BillingServiceClient
	response *pb.ProcessAlipayNotificationResponse
	err      error
}

func (f webhookBillingClient) ProcessAlipayNotification(context.Context, *pb.ProcessAlipayNotificationRequest, ...grpc.CallOption) (*pb.ProcessAlipayNotificationResponse, error) {
	return f.response, f.err
}

func TestAlipayWebhookAcknowledgesOnlyDurablyAcceptedNotifications(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		response   *pb.ProcessAlipayNotificationResponse
		err        error
		wantStatus int
		wantBody   string
	}{
		{name: "accepted", response: &pb.ProcessAlipayNotificationResponse{Accepted: true}, wantStatus: http.StatusOK, wantBody: "success"},
		{name: "invalid", err: status.Error(codes.InvalidArgument, "bad signature"), wantStatus: http.StatusBadRequest, wantBody: "failure"},
		{name: "transient", err: status.Error(codes.Internal, "database unavailable"), wantStatus: http.StatusServiceUnavailable, wantBody: "failure"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clients := &rpc.Clients{Billing: webhookBillingClient{response: test.response, err: test.err}}
			handler := NewAlipayWebhookHandler(clients, "http://localhost:5173/hr/billing", "http://localhost:5174/billing")
			router := gin.New()
			router.POST("/notify", handler.Notify)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/notify", strings.NewReader("out_trade_no=P1&trade_status=TRADE_SUCCESS"))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			router.ServeHTTP(recorder, request)
			if recorder.Code != test.wantStatus || recorder.Body.String() != test.wantBody {
				t.Fatalf("response = %d %q, want %d %q", recorder.Code, recorder.Body.String(), test.wantStatus, test.wantBody)
			}
		})
	}
}

func TestSaveBillingPriceRequestAcceptsProtoJSONIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{
			name: "unchanged protojson values",
			body: `{"product_id":"42","price_version_id":"7","billing_term":"monthly","amount_fen":"3900","included_credits":"100","publish":true}`,
		},
		{
			name: "numeric IDs sent directly by clients",
			body: `{"product_id":42,"price_version_id":7,"billing_term":"monthly","amount_fen":3900,"included_credits":100,"publish":true}`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Request = httptest.NewRequest("POST", "/api/v1/platform/billing/prices", strings.NewReader(test.body))
			context.Request.Header.Set("Content-Type", "application/json")

			var request saveBillingPriceRequest
			if err := context.ShouldBindJSON(&request); err != nil {
				t.Fatalf("bind request: %v", err)
			}
			if request.ProductID != 42 || request.PriceVersionID != 7 {
				t.Fatalf("ids = (%d, %d), want (42, 7)", request.ProductID, request.PriceVersionID)
			}
			if request.BillingTerm != "monthly" || request.AmountFen != 3900 || request.IncludedCredits != 100 || !request.Publish {
				t.Fatalf("unexpected request: %+v", request)
			}
		})
	}
}

func TestCreateBillingOrderRequestAcceptsProtoJSONPriceVersionID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "protojson string ID", body: `{"price_version_id":"5","order_type":"subscribe","idempotency_key":"order-1","replace_pending_order":true}`},
		{name: "numeric ID", body: `{"price_version_id":5,"order_type":"subscribe","idempotency_key":"order-2"}`},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Request = httptest.NewRequest("POST", "/api/v1/hr/billing/orders", strings.NewReader(test.body))
			context.Request.Header.Set("Content-Type", "application/json")

			var request createBillingOrderRequest
			if err := context.ShouldBindJSON(&request); err != nil {
				t.Fatalf("bind request: %v", err)
			}
			if request.PriceVersionID != 5 || request.OrderType != "subscribe" || request.IdempotencyKey == "" {
				t.Fatalf("unexpected request: %+v", request)
			}
			if test.name == "protojson string ID" && !request.ReplacePendingOrder {
				t.Fatal("replace_pending_order was not bound")
			}
		})
	}
}

func TestCreateBillingOrderRequestRejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "missing price version", body: `{"order_type":"subscribe","idempotency_key":"order-1"}`},
		{name: "missing order type", body: `{"price_version_id":"5","idempotency_key":"order-1"}`},
		{name: "missing idempotency key", body: `{"price_version_id":"5","order_type":"subscribe"}`},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Request = httptest.NewRequest("POST", "/api/v1/hr/billing/orders", strings.NewReader(test.body))
			context.Request.Header.Set("Content-Type", "application/json")

			var request createBillingOrderRequest
			if err := context.ShouldBindJSON(&request); err == nil {
				t.Fatal("expected missing required field to fail validation")
			}
		})
	}
}
