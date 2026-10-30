package routes

import (
	"net/http"
	"sort"
	"strconv"

	"maps/backend/internal/engine"
	"maps/backend/internal/types"
	"maps/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

func init() {
	Register(ApiRoute{
		Method: http.MethodGet,
		Path:   "/stops/autocomplete",
		Handler: func(c *gin.Context, e engine.Engine) {
			q := c.Query("q")

			sMaxStops := c.Query("maxstops")
			maxStops := 20
			if sMaxStops != "" {
				maxStops, _ = strconv.Atoi(sMaxStops)
			}

			if len(q) <= 2 {
				c.JSON(http.StatusOK, gin.H{
					"total": 0,
					"stops": []types.ParentsStation{},
				})
				return
			}

			sLat, sLon := c.Query("lat"), c.Query("lon")
			hasCoords := false
			var lat, lon float64
			if sLat != "" || sLon != "" {
				if sLat == "" || sLon == "" {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "need to have both coordinates",
					})
					return
				}

				var errLat, errLon error
				lat, errLat = strconv.ParseFloat(sLat, 64)
				lon, errLon = strconv.ParseFloat(sLon, 64)
				if errLat != nil || errLon != nil {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "invalid lat or lon coordinate",
					})
					return
				}
				hasCoords = true
			}

			stops := e.SearchParentsStopsByName(q)

			if hasCoords && len(stops) > 0 {
				sort.SliceStable(stops, func(i, j int) bool {
					hasI := stops[i].Lat != 0 || stops[i].Lon != 0
					hasJ := stops[j].Lat != 0 || stops[j].Lon != 0
					if hasI && !hasJ {
						return true
					}
					if !hasI && hasJ {
						return false
					}
					if !hasI && !hasJ {
						return false
					}
					dI := utils.CoordsGetDistance(stops[i].Lon, stops[i].Lat, lon, lat)
					dJ := utils.CoordsGetDistance(stops[j].Lon, stops[j].Lat, lon, lat)
					return dI < dJ
				})
			}

			if maxStops > 0 && len(stops) > maxStops {
				stops = stops[:maxStops]
			}

			c.JSON(http.StatusOK, gin.H{
				"total": len(stops),
				"stops": stops,
			})
		},
	})
}
