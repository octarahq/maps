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

type nextDepartureDetail struct {
	WaitSec             int    `json:"wait_sec"`
	DepSec              int    `json:"dep_sec"`
	ScheduledDepSec     int    `json:"scheduled_dep_sec"`
	DelaySec            int    `json:"delay_sec"`
	DelayMinutes        int    `json:"delay_minutes"`
	IsDelayed           bool   `json:"is_delayed"`
	TimeString          string `json:"time_string"`
	ScheduledTimeString string `json:"scheduled_time_string"`
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

func dedupSortedDetails(in []nextDepartureDetail) []nextDepartureDetail {
	if len(in) <= 1 {
		return in
	}
	out := make([]nextDepartureDetail, 0, len(in))
	for _, v := range in {
		if len(out) == 0 || out[len(out)-1].WaitSec != v.WaitSec {
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

			withDetails := c.Query("details") == "true" || c.Query("details") == "1"

			activeServices := e.GetActiveServices(targetTime)
			ad := e.GetActiveDelays(targetTime)
			currSec := uint32(targetTime.Hour()*3600 + targetTime.Minute()*60 + targetTime.Second())

			routes := utils.IndexRoutesByID(e.Net.Routes)
			res := make(map[string]map[string][]int)
			resDetails := make(map[string]map[string][]nextDepartureDetail)

			const maxWindowSec = 4 * 3600

			for _, req := range reqs {
				edges := collectEdgesForStop(&e, req.stopID, req.cleanSID)
				routeDepartures := make(map[string][]int)
				routeDetails := make(map[string][]nextDepartureDetail)

				for i := range edges {
					edge := &edges[i]
					if edge.RID == "" {
						continue
					}
					edgeCleanRID := utils.IdRemoveIDFMPrefix(edge.RID)
					if edgeCleanRID != req.cleanRID && edge.RID != req.routeID && edge.RID != req.cleanRID {
						continue
					}

					rtScheds := e.FindNextDeparturesRealtime(edge, currSec, limit, maxWindowSec, activeServices, ad)
					if len(rtScheds) == 0 {
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

					for _, rs := range rtScheds {
						waitSec := int(rs.RealDep) - int(currSec)
						if waitSec < 0 {
							waitSec = 0
						}
						routeDepartures[headsign] = append(routeDepartures[headsign], waitSec)

						delayMin := (rs.DepDelaySec + 30) / 60
						if delayMin < 0 {
							delayMin = 0
						}
						schedDep := int(rs.Sched.Dep)
						realDep := int(rs.RealDep)
						routeDetails[headsign] = append(routeDetails[headsign], nextDepartureDetail{
							WaitSec:             waitSec,
							DepSec:              realDep,
							ScheduledDepSec:     schedDep,
							DelaySec:            rs.DepDelaySec,
							DelayMinutes:        delayMin,
							IsDelayed:           rs.DepDelaySec >= 60,
							TimeString:          fmt.Sprintf("%02d:%02d", (realDep/3600)%24, (realDep%3600)/60),
							ScheduledTimeString: fmt.Sprintf("%02d:%02d", (schedDep/3600)%24, (schedDep%3600)/60),
						})
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

				for h, details := range routeDetails {
					sort.Slice(details, func(i, j int) bool {
						return details[i].WaitSec < details[j].WaitSec
					})
					details = dedupSortedDetails(details)
					if len(details) > limit {
						details = details[:limit]
					}
					routeDetails[h] = details
				}

				key := fmt.Sprintf("%s:%s", req.cleanSID, req.cleanRID)
				res[key] = routeDepartures
				resDetails[key] = routeDetails

				if req.rawKey != key && req.rawKey != "" {
					res[req.rawKey] = routeDepartures
					resDetails[req.rawKey] = routeDetails
				}
			}

			if withDetails {
				c.JSON(http.StatusOK, resDetails)
				return
			}
			c.JSON(http.StatusOK, res)
		},
	})
}
