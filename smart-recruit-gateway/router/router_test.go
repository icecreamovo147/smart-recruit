package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"smart-recruit-gateway/config"
	"smart-recruit-gateway/rpc"
	"smart-recruit-platform-go/i18n"
)

func TestSetupDoesNotPanicWithApplicationIntelligenceRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, limiters := Setup(config.Config{}, &rpc.Clients{}, nil)
	if limiters != nil {
		limiters.Close()
	}
}

func TestRuntimeConfigExposesConfiguredLocaleWithoutAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if err := i18n.Configure("en-US"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = i18n.Configure("zh-CN") })

	engine, limiters := Setup(config.Config{}, &rpc.Clients{}, nil)
	if limiters != nil {
		defer limiters.Close()
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/public/runtime-config", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var body struct {
		MessageKey string `json:"message_key"`
		Msg        string `json:"msg"`
		Data       struct {
			Locale string `json:"locale"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.MessageKey != "common.success" || body.Msg != "Success" || body.Data.Locale != "en-US" {
		t.Fatalf("unexpected runtime config: %#v", body)
	}
}
