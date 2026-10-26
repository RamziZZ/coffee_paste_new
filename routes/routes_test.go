package routes

import (
	"testing"

	"coffee_paste_backend/config"

	"github.com/gin-gonic/gin"
)

// TestSetupRoutesNoPanic memastikan semua route bisa didaftarkan
// tanpa konflik (static vs param) di tree Gin.
func TestSetupRoutesNoPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := &config.Config{JWTSecret: "test-secret"}
	SetupRoutes(r, nil, cfg)
}
