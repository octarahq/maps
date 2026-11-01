package routes

import (
	"maps/backend/internal/engine"
	"maps/backend/internal/utils"
	"net/http"
	"time"

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
			targetTime := time.Now()
			if timeParam := c.Query("time"); timeParam != "" {
				if t, err := time.Parse(time.RFC3339, timeParam); err == nil {
					targetTime = t
				} else if t, err := time.Parse("2006-01-02T15:04:05", timeParam); err == nil {
					targetTime = t
				} else if t, err := time.Parse("2006-01-02T15:04", timeParam); err == nil {
					targetTime = t
				} else if t, err := time.Parse("15:04", timeParam); err == nil {
					now := time.Now()
					targetTime = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
				}
			}

			rid := c.Param("rid")
			rDepartures := e.GetNextDeparturesForRoute(rid, targetTime)

			c.JSON(http.StatusOK, rDepartures)
		},
	})
}
