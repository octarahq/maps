package engine

import (
	"fmt"
	"sort"
	"time"

	"maps/backend/internal/types"
	"maps/backend/internal/utils"
)

func (e *Engine) GetNextDeparturesForStop(stopID string, targetTime time.Time) []types.DepartureInfo {
	targetTime = utils.InParis(targetTime)

	activeServices := e.GetActiveServices(targetTime)
	currSec := uint32(targetTime.Hour()*3600 + targetTime.Minute()*60 + targetTime.Second())

	year, month, day := targetTime.Date()
	baseDate := time.Date(year, month, day, 0, 0, 0, 0, targetTime.Location())

	cleanSID := utils.IdRemoveIDFMPrefix(stopID)
	edges := e.Graph[stopID]
	if len(edges) == 0 {
		edges = e.Graph["IDFM:"+stopID]
	}
	if len(edges) == 0 {
		edges = e.Graph["IDFM:"+cleanSID]
	}
	if len(edges) == 0 {
		edges = e.Graph[cleanSID]
	}

	hasTransit := false
	for _, edge := range edges {
		if edge.RID != "" {
			hasTransit = true
			break
		}
	}

	if !hasTransit {
		if parent, ok := e.SearchParentsStopById(stopID); ok && len(parent.ChildrenStops) > 0 {
			allResults := make([]types.DepartureInfo, 0)
			for _, childSID := range parent.ChildrenStops {
				allResults = append(allResults, e.GetNextDeparturesForStop(childSID, targetTime)...)
			}
			sort.Slice(allResults, func(i, j int) bool {
				return allResults[i].DepSec < allResults[j].DepSec
			})
			return allResults
		}
	}

	results := make([]types.DepartureInfo, 0)
	routes := utils.IndexRoutesByID(e.Net.Routes)

	for _, edge := range edges {
		if edge.RID == "" {
			continue
		}

		sched, ok := utils.FindNextDeparture(edge.Schedules, currSec, activeServices)
		if !ok {
			continue
		}

		route := routes[edge.RID]
		name := route.SName
		if name == "" {
			name = route.LName
		}

		waitSec := int(sched.Dep) - int(currSec)
		waitMin := float64(waitSec) / 60.0
		if waitMin < 0 {
			waitMin = 0
		}

		cleanRID := utils.IdRemoveIDFMPrefix(route.RID)
		if cleanRID == "" {
			cleanRID = utils.IdRemoveIDFMPrefix(edge.RID)
		}

		results = append(results, types.DepartureInfo{
			RID:         cleanRID,
			RName:       name,
			RColor:      route.Color,
			RType:       route.RType,
			Picto:       route.Picto,
			Headsign:    edge.Headsign,
			DepSec:      int(sched.Dep),
			Time:        baseDate.Add(time.Duration(sched.Dep) * time.Second),
			TimeString:  fmt.Sprintf("%02d:%02d", sched.Dep/3600, (sched.Dep%3600)/60),
			WaitMinutes: waitMin,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].DepSec < results[j].DepSec
	})

	return results
}

func (e *Engine) GetNextDeparturesForRoute(rid string, targetTime time.Time) map[string][]types.DepartureInfo {
	targetTime = utils.InParis(targetTime)

	activeServices := e.GetActiveServices(targetTime)
	currSec := uint32(targetTime.Hour()*3600 + targetTime.Minute()*60 + targetTime.Second())
	year, month, day := targetTime.Date()
	baseDate := time.Date(year, month, day, 0, 0, 0, 0, targetTime.Location())

	routes := utils.IndexRoutesByID(e.Net.Routes)
	cleanRID := utils.IdRemoveIDFMPrefix(rid)
	route, ok := routes[rid]
	if !ok {
		route, ok = routes[cleanRID]
	}
	if !ok {
		route, ok = routes["IDFM:"+cleanRID]
	}

	name := route.SName
	if name == "" {
		name = route.LName
	}
	cleanRouteRID := utils.IdRemoveIDFMPrefix(route.RID)
	if cleanRouteRID == "" {
		cleanRouteRID = cleanRID
	}

	results := make(map[string][]types.DepartureInfo)
	for stopID, edges := range e.Graph {
		for _, edge := range edges {
			edgeCleanRID := utils.IdRemoveIDFMPrefix(edge.RID)
			if edge.RID != rid && edgeCleanRID != cleanRID && edgeCleanRID != cleanRouteRID {
				continue
			}

			sched, ok := utils.FindNextDeparture(edge.Schedules, currSec, activeServices)
			if !ok {
				continue
			}

			waitSec := int(sched.Dep) - int(currSec)
			waitMin := float64(waitSec) / 60.0
			if waitMin < 0 {
				waitMin = 0
			}

			cleanStopID := utils.IdRemoveIDFMPrefix(stopID)
			results[cleanStopID] = append(results[cleanStopID], types.DepartureInfo{
				RID:         cleanRouteRID,
				RName:       name,
				RColor:      route.Color,
				RType:       route.RType,
				Picto:       route.Picto,
				Headsign:    edge.Headsign,
				DepSec:      int(sched.Dep),
				Time:        baseDate.Add(time.Duration(sched.Dep) * time.Second),
				TimeString:  fmt.Sprintf("%02d:%02d", sched.Dep/3600, (sched.Dep%3600)/60),
				WaitMinutes: waitMin,
			})
		}
	}

	for stopID := range results {
		sort.Slice(results[stopID], func(i, j int) bool {
			return results[stopID][i].DepSec < results[stopID][j].DepSec
		})
	}

	return results
}
