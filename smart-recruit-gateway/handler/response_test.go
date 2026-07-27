package handler

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"smart-recruit-platform-go/i18n"
)

func TestPublicErrorPreservesFailedPreconditionMessage(t *testing.T) {
	const message = "订单已超过支付时限并关闭，请重新创建订单"
	info := PublicError(status.Error(codes.FailedPrecondition, message))
	if info.Code != 409 || info.Msg != message {
		t.Fatalf("PublicError() = %#v, want code 409 and payment guidance", info)
	}
}

func TestPublicErrorMapsInsufficientAICredits(t *testing.T) {
	info := PublicError(status.Error(codes.ResourceExhausted, "insufficient_credits"))
	if info.Code != 40201 || info.Msg != "ai.insufficient_credits" {
		t.Fatalf("PublicError() = %#v, want AI credit purchase guidance", info)
	}
}

func TestResponseEnvelopeUsesStableKeyAndConfiguredLocale(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if err := i18n.Configure("en-US"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = i18n.Configure("zh-CN") })

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	BadRequest(context, "上游返回的原始中文错误")

	var body struct {
		Code       int32  `json:"code"`
		MessageKey string `json:"message_key"`
		Msg        string `json:"msg"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 400 || body.MessageKey != "common.invalid_request" {
		t.Fatalf("unexpected envelope: %#v", body)
	}
	if body.Msg != i18n.T("common.invalid_request") {
		t.Fatalf("unexpected English message: %q", body.Msg)
	}
}

func TestLocalizedMessageDoesNotExposeUnknownUpstreamText(t *testing.T) {
	if err := i18n.Configure("zh-CN"); err != nil {
		t.Fatal(err)
	}
	key, message := LocalizedMessage(500, "vendor secret failure detail")
	if key != "common.operation_failed" || message != i18n.T(key) {
		t.Fatalf("LocalizedMessage() = %q, %q", key, message)
	}
}
