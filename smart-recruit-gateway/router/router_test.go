package router

import (
	"testing"

	"github.com/gin-gonic/gin"

	"smart-recruit-gateway/config"
	"smart-recruit-gateway/rpc"
)

func TestSetupDoesNotPanicWithApplicationIntelligenceRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, limiters := Setup(config.Config{}, &rpc.Clients{}, nil)
	if limiters != nil {
		limiters.Close()
	}
}
