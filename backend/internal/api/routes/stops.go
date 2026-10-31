package routes

import (
	"net/http"
	"sort"
	"strconv"
	"time"

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

	Register(ApiRoute{
		Method: http.MethodGet,
		Path:   "/stops/:sid",
		Handler: func(c *gin.Context, e engine.Engine) {
			sid := c.Param("sid")
			stops := utils.IndexStopsByID(e.Net.Stops)

			stop, ok := stops[sid]
			if !ok {
				stop, ok = stops[utils.IdRemoveIDFMPrefix(sid)]
			}
			if !ok {
				stop, ok = e.SearchStopById(sid)
			}
			if !ok {
				c.Status(http.StatusNotFound)
				return
			}

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

			departures := e.GetNextDeparturesForStop(sid, targetTime)
			if len(departures) == 0 && stop.ParentStation != "" {
				departures = e.GetNextDeparturesForStop(stop.ParentStation, targetTime)
			}

			c.JSON(http.StatusOK, gin.H{
				"stop":       stop,
				"departures": departures,
			})
		},
	})

	Register(ApiRoute{
		Method: http.MethodGet,
		Path:   "/stops/nearby",
		Handler: func(c *gin.Context, e engine.Engine) {
			sLat, sLon := c.Query("lat"), c.Query("lon")
			if sLat == "" || sLon == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "need lat and lon query",
				})
				return
			}

			lat, err := strconv.ParseFloat(sLat, 64)
			lon, err := strconv.ParseFloat(sLon, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "bad coord format",
				})
				return
			}

			maxK := 10
			if c.Query("max") != "" {
				maxK, err = strconv.Atoi(c.Query("max"))
				if err != nil {
					maxK = 10
				}
			}

			nStops := e.SearchStopsByNeerCord(lon, lat, maxK)

			c.JSON(http.StatusOK, gin.H{
				"stops": nStops,
			})
		},
	})
}
