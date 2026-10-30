package routes

import (
	"maps/backend/internal/engine"
	"net/http"

	"github.com/gin-gonic/gin"
)

func init() {
	Register(ApiRoute{
		Method: http.MethodGet,
		Path:   "/health",
		Handler: func(c *gin.Context, e engine.Engine) {
			c.JSON(http.StatusOK, gin.H{
				"status": "ok",
				"stats": gin.H{
					"stops":       len(e.Net.Stops),
					"routes":      len(e.Net.Routes),
					"trips":       len(e.Net.Trips),
					"stops_times": len(e.Net.StopTimes),
				},
			})
		},
	})
}
