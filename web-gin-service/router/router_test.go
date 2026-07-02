package router

import (
	"testing"

	"github.com/gin-gonic/gin"

	"web-gin-service/config"
	"web-gin-service/rpc"
)

func TestSetupDoesNotPanicWithApplicationIntelligenceRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, limiters := Setup(config.Config{}, &rpc.Clients{}, nil)
	if limiters != nil {
		limiters.Close()
	}
}
