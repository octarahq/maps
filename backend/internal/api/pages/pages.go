package pages

import (
	_ "embed"
	"net/http"

	"maps/backend/internal/engine"

	"github.com/gin-gonic/gin"
)

//go:embed index.html
var indexHTML string

func Register(router *gin.Engine, e engine.Engine) {
	handler := func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, indexHTML)
	}

	router.GET("/", handler)
}
