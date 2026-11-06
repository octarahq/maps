package routes

import (
	"maps/backend/internal/engine"
	"maps/backend/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func init() {
	Register(ApiRoute{
		Method: http.MethodGet,
		Path:   "/debug/disruptions",
		Handler: func(c *gin.Context, e engine.Engine) {
			slType := c.Query("type")

			now := utils.NowInParis()
			diruptions := e.GetActiveDisruptions(now)
			switch slType {
			case "lines":
				var lines []string
				for line, d := range diruptions.BlockedLineStops {
					if d {
						lines = append(lines, line)
					}
				}

				c.JSON(http.StatusOK, gin.H{
					"lines": lines,
				})
			default:
				c.Status(http.StatusNotFound)
			}
		},
	})
}
