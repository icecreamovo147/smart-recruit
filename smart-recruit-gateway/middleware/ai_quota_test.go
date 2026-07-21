package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAIQuotaRequestConsumed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name   string
		status int
		mark   func(*gin.Context)
		want   bool
	}{
		{name: "successful ordinary handler", status: http.StatusOK, want: true},
		{name: "failed ordinary handler", status: http.StatusTooManyRequests, want: false},
		{name: "SSE admitted before later error", status: http.StatusOK, mark: MarkAIQuotaConsumed, want: true},
		{name: "SSE rejected before admission", status: http.StatusOK, mark: MarkAIQuotaFailed, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Status(tt.status)
			if tt.mark != nil {
				tt.mark(ctx)
			}
			if got := aiQuotaRequestConsumed(ctx); got != tt.want {
				t.Fatalf("aiQuotaRequestConsumed() = %v, want %v", got, tt.want)
			}
		})
	}
}
