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
		Path:   "/routes/:rid",
		Handler: func(c *gin.Context, e engine.Engine) {
			rid := c.Param("rid")

			routes := utils.IndexRoutesByID(e.Net.Routes)
			route, ok := routes[rid]
			if !ok {
				c.Status(http.StatusNotFound)
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"route": route,
			})
		},
	})
}
