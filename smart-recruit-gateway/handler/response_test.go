package handler

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPublicErrorPreservesFailedPreconditionMessage(t *testing.T) {
	t.Parallel()

	const message = "订单已超过支付时限并关闭，请重新创建订单"
	info := PublicError(status.Error(codes.FailedPrecondition, message))
	if info.Code != 409 || info.Msg != message {
		t.Fatalf("PublicError() = %#v, want code 409 and payment guidance", info)
	}
}
