package routes

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"maps/backend/internal/engine"
	"maps/backend/internal/types"
	"maps/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type stopRouteReq struct {
	rawKey   string
	stopID   string
	routeID  string
	cleanSID string
	cleanRID string
}

func parseStopRoutePair(raw string) (stopRouteReq, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return stopRouteReq{}, false
	}

	var sid, rid string
	if parts := strings.Split(raw, ";"); len(parts) == 2 {
		sid, rid = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	} else {
		clean := utils.IdRemoveIDFMPrefix(raw)
		if parts := strings.Split(clean, ":"); len(parts) == 2 {
			sid, rid = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		} else if parts := strings.Split(raw, ":"); len(parts) == 2 {
			sid, rid = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		}
	}

	if sid == "" || rid == "" {
		return stopRouteReq{}, false
	}

	return stopRouteReq{
		rawKey:   raw,
		stopID:   sid,
		routeID:  rid,
		cleanSID: utils.IdRemoveIDFMPrefix(sid),
		cleanRID: utils.IdRemoveIDFMPrefix(rid),
	}, true
}

func getDirectEdges(e *engine.Engine, stopID, cleanSID string) []types.Edge {
	if edges := e.GetEdgesForStop(stopID); len(edges) > 0 {
		return edges
	}
	if cleanSID != "" && cleanSID != stopID {
		if edges := e.GetEdgesForStop(cleanSID); len(edges) > 0 {
			return edges
		}
	}
	return nil
}

func collectEdgesForStop(e *engine.Engine, stopID, cleanSID string) []types.Edge {
	edges := getDirectEdges(e, stopID, cleanSID)

	var children []string
	if parent, ok := e.SearchParentsStopById(stopID); ok && len(parent.ChildrenStops) > 0 {
		children = parent.ChildrenStops
	} else if parent, ok := e.SearchParentsStopById(cleanSID); ok && len(parent.ChildrenStops) > 0 {
		children = parent.ChildrenStops
	} else if stop, ok := e.SearchStopById(stopID); ok && stop.ParentStation != "" {
		cleanParent := utils.IdRemoveIDFMPrefix(stop.ParentStation)
		return collectEdgesForStop(e, stop.ParentStation, cleanParent)
	} else if stop, ok := e.SearchStopById(cleanSID); ok && stop.ParentStation != "" {
		cleanParent := utils.IdRemoveIDFMPrefix(stop.ParentStation)
		return collectEdgesForStop(e, stop.ParentStation, cleanParent)
	}

	for _, childSID := range children {
		if utils.IdRemoveIDFMPrefix(childSID) == cleanSID || childSID == stopID {
			continue
		}
		cleanChild := utils.IdRemoveIDFMPrefix(childSID)
		edges = append(edges, getDirectEdges(e, childSID, cleanChild)...)
	}

	return edges
}

func dedupSortedInts(in []int) []int {
	if len(in) <= 1 {
		return in
	}
	out := make([]int, 0, len(in))
	for _, v := range in {
		if len(out) == 0 || out[len(out)-1] != v {
			out = append(out, v)
		}
	}
	return out
}

func init() {
	Register(ApiRoute{
		Method: http.MethodGet,
		Path:   "/nextdepartures",
		Handler: func(c *gin.Context, e engine.Engine) {
			var stopTokens []string
			for _, s := range c.QueryArray("stops") {
				for _, part := range strings.Split(s, ",") {
					part = strings.TrimSpace(part)
					if part != "" {
						stopTokens = append(stopTokens, part)
					}
				}
			}
			if len(stopTokens) == 0 {
				if s := c.Query("stops"); s != "" {
					for _, part := range strings.Split(s, ",") {
						part = strings.TrimSpace(part)
						if part != "" {
							stopTokens = append(stopTokens, part)
						}
					}
				}
			}

			if len(stopTokens) == 0 {
				c.JSON(http.StatusOK, gin.H{})
				return
			}

			var reqs []stopRouteReq
			for _, token := range stopTokens {
				if req, ok := parseStopRoutePair(token); ok {
					reqs = append(reqs, req)
				}
			}

			if len(reqs) == 0 {
				c.JSON(http.StatusOK, gin.H{})
				return
			}

			targetTime := utils.NowInParis()
			if timeParam := c.Query("time"); timeParam != "" {
				if t, ok := utils.ParseCustomTimeInParis(timeParam); ok {
					targetTime = t
				}
			}

			limit := 3
			if sLimit := c.Query("limit"); sLimit != "" {
				if l, err := strconv.Atoi(sLimit); err == nil && l > 0 && l <= 10 {
					limit = l
				}
			}

			activeServices := e.GetActiveServices(targetTime)
			currSec := uint32(targetTime.Hour()*3600 + targetTime.Minute()*60 + targetTime.Second())

			routes := utils.IndexRoutesByID(e.Net.Routes)
			res := make(map[string]map[string][]int)

			const maxWindowSec = 4 * 3600

			for _, req := range reqs {
				edges := collectEdgesForStop(&e, req.stopID, req.cleanSID)
				routeDepartures := make(map[string][]int)

				for _, edge := range edges {
					if edge.RID == "" {
						continue
					}
					edgeCleanRID := utils.IdRemoveIDFMPrefix(edge.RID)
					if edgeCleanRID != req.cleanRID && edge.RID != req.routeID && edge.RID != req.cleanRID {
						continue
					}

					scheds := utils.FindNextDepartures(edge.Schedules, currSec, limit, maxWindowSec, activeServices)
					if len(scheds) == 0 {
						continue
					}

					headsign := strings.TrimSpace(edge.Headsign)
					if headsign == "" {
						if r, ok := routes[edge.RID]; ok {
							headsign = r.LName
							if headsign == "" {
								headsign = r.SName
							}
						}
					}
					if headsign == "" {
						headsign = "Direction inconnue"
					}

					for _, s := range scheds {
						waitSec := int(s.Dep) - int(currSec)
						if waitSec < 0 {
							waitSec = 0
						}
						routeDepartures[headsign] = append(routeDepartures[headsign], waitSec)
					}
				}

				for h, times := range routeDepartures {
					sort.Ints(times)
					times = dedupSortedInts(times)
					if len(times) > limit {
						times = times[:limit]
					}
					routeDepartures[h] = times
				}

				key := fmt.Sprintf("%s:%s", req.cleanSID, req.cleanRID)
				res[key] = routeDepartures

				if req.rawKey != key && req.rawKey != "" {
					res[req.rawKey] = routeDepartures
				}
			}

			c.JSON(http.StatusOK, res)
		},
	})
}
