package grpc

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"smart-recruit-billing-service/internal/application/service"
)

func TestCreateBillingOrderError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantCode    codes.Code
		wantMessage string
	}{
		{
			name:        "scheduled renewal is a business conflict",
			err:         service.ErrSubscriptionRenewalScheduled,
			wantCode:    codes.FailedPrecondition,
			wantMessage: "当前套餐已完成续费，下一周期套餐将在生效日自动启用，无需重复购买",
		},
		{
			name:        "wrapped scheduled renewal keeps its classification",
			err:         errors.Join(errors.New("create order"), service.ErrSubscriptionRenewalScheduled),
			wantCode:    codes.FailedPrecondition,
			wantMessage: "当前套餐已完成续费，下一周期套餐将在生效日自动启用，无需重复购买",
		},
		{
			name:        "validation error remains invalid argument",
			err:         errors.New("invalid billing order type"),
			wantCode:    codes.InvalidArgument,
			wantMessage: "invalid billing order type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := status.Convert(createBillingOrderError(tt.err))
			if got.Code() != tt.wantCode {
				t.Fatalf("code = %s, want %s", got.Code(), tt.wantCode)
			}
			if got.Message() != tt.wantMessage {
				t.Fatalf("message = %q, want %q", got.Message(), tt.wantMessage)
			}
		})
	}
}
