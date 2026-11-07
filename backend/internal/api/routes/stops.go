package routes

import (
	"cmp"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"

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

			for i := range stops {
				stops[i].PID = utils.IdRemoveIDFMPrefix(stops[i].PID)
				for j := range stops[i].ChildrenStops {
					stops[i].ChildrenStops[j] = utils.IdRemoveIDFMPrefix(stops[i].ChildrenStops[j])
				}
				for j := range stops[i].Routes {
					stops[i].Routes[j].RID = utils.IdRemoveIDFMPrefix(stops[i].Routes[j].RID)
				}
			}

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

			routesMap := utils.IndexRoutesByID(e.Net.Routes)
			cleanSID := utils.IdRemoveIDFMPrefix(stop.SID)

			var targetStopIDs []string
			if stop.ParentStation == "" {
				seenChildren := make(map[string]bool)
				if ps, ok := e.SearchParentsStopById(stop.SID); ok {
					for _, c := range ps.ChildrenStops {
						cleanC := utils.IdRemoveIDFMPrefix(c)
						if cleanC != "" && !seenChildren[cleanC] {
							seenChildren[cleanC] = true
							targetStopIDs = append(targetStopIDs, cleanC)
						}
					}
				}
				for _, s := range e.Net.Stops {
					if s.ParentStation != "" && utils.IdRemoveIDFMPrefix(s.ParentStation) == cleanSID {
						cleanC := utils.IdRemoveIDFMPrefix(s.SID)
						if cleanC != "" && !seenChildren[cleanC] {
							seenChildren[cleanC] = true
							targetStopIDs = append(targetStopIDs, cleanC)
						}
					}
				}
				if len(targetStopIDs) == 0 {
					targetStopIDs = append(targetStopIDs, cleanSID)
				}
			} else {
				targetStopIDs = append(targetStopIDs, cleanSID)
			}

			seenRouteIDs := make(map[string]bool)
			routes := make([]types.Route, 0)

			addRoute := func(r types.Route) {
				cleanRID := utils.IdRemoveIDFMPrefix(r.RID)
				if cleanRID == "" || seenRouteIDs[cleanRID] {
					return
				}
				seenRouteIDs[cleanRID] = true
				cleanRoute := r
				cleanRoute.RID = cleanRID
				routes = append(routes, cleanRoute)
			}

			if stop.ParentStation == "" {
				if ps, ok := e.SearchParentsStopById(stop.SID); ok {
					for _, pr := range ps.Routes {
						if r, found := routesMap[pr.RID]; found {
							addRoute(r)
						} else if r, found := routesMap[utils.IdRemoveIDFMPrefix(pr.RID)]; found {
							addRoute(r)
						}
					}
				}
			}

			for _, sidTarget := range targetStopIDs {
				for _, edge := range e.GetEdgesForStop(sidTarget) {
					if edge.RID == "" {
						continue
					}
					if r, found := routesMap[edge.RID]; found {
						addRoute(r)
					} else if r, found := routesMap[utils.IdRemoveIDFMPrefix(edge.RID)]; found {
						addRoute(r)
					}
				}
				for _, edge := range e.GetReverseEdgesForStop(sidTarget) {
					if edge.RID == "" {
						continue
					}
					if r, found := routesMap[edge.RID]; found {
						addRoute(r)
					} else if r, found := routesMap[utils.IdRemoveIDFMPrefix(edge.RID)]; found {
						addRoute(r)
					}
				}
			}

			slices.SortFunc(routes, func(a, b types.Route) int {
				if a.RType != b.RType {
					return cmp.Compare(a.RType, b.RType)
				}
				nameA := a.SName
				if nameA == "" {
					nameA = a.LName
				}
				nameB := b.SName
				if nameB == "" {
					nameB = b.LName
				}
				return strings.Compare(nameA, nameB)
			})

			cleanStop := stop
			cleanStop.SID = utils.IdRemoveIDFMPrefix(stop.SID)
			cleanStop.ParentStation = utils.IdRemoveIDFMPrefix(stop.ParentStation)

			targetTime := utils.NowInParis()
			if timeParam := c.Query("time"); timeParam != "" {
				if t, ok := utils.ParseCustomTimeInParis(timeParam); ok {
					targetTime = t
				}
			}

			departures := e.GetNextDeparturesForStop(sid, targetTime)
			if len(departures) == 0 && stop.ParentStation != "" {
				departures = e.GetNextDeparturesForStop(stop.ParentStation, targetTime)
			}

			c.JSON(http.StatusOK, gin.H{
				"stop":       cleanStop,
				"departures": departures,
				"routes":     routes,
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
			for i := range nStops {
				nStops[i].SID = utils.IdRemoveIDFMPrefix(nStops[i].SID)
				nStops[i].ParentStation = utils.IdRemoveIDFMPrefix(nStops[i].ParentStation)
			}

			c.JSON(http.StatusOK, gin.H{
				"stops": nStops,
			})
		},
	})
}
