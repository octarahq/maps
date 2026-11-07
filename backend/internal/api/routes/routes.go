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
				route, ok = routes[utils.IdRemoveIDFMPrefix(rid)]
			}
			if !ok {
				c.Status(http.StatusNotFound)
				return
			}

			cleanRoute := route
			cleanRoute.RID = utils.IdRemoveIDFMPrefix(route.RID)

			c.JSON(http.StatusOK, gin.H{
				"route": cleanRoute,
			})
		},
	})

	Register(ApiRoute{
		Method: http.MethodGet,
		Path:   "/routes/:rid/departures",
		Handler: func(c *gin.Context, e engine.Engine) {
			targetTime := utils.NowInParis()
			if timeParam := c.Query("time"); timeParam != "" {
				if t, ok := utils.ParseCustomTimeInParis(timeParam); ok {
					targetTime = t
				}
			}

			rid := c.Param("rid")
			rDepartures := e.GetNextDeparturesForRoute(rid, targetTime)

			c.JSON(http.StatusOK, rDepartures)
		},
	})
}
