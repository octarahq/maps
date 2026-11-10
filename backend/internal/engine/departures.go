package engine

import (
	"fmt"
	"sort"
	"time"

	"maps/backend/internal/types"
	"maps/backend/internal/utils"
)

func buildDepartureInfo(
	cleanRID string,
	name string,
	route types.Route,
	headsign string,
	rtSched RealtimeSchedule,
	currSec uint32,
	baseDate time.Time,
) types.DepartureInfo {
	waitSec := int(rtSched.RealDep) - int(currSec)
	waitMin := float64(waitSec) / 60.0
	if waitMin < 0 {
		waitMin = 0
	}

	schedDep := int(rtSched.Sched.Dep)
	realDep := int(rtSched.RealDep)
	delaySec := rtSched.DepDelaySec
	isDelayed := delaySec >= 60
	delayMin := (delaySec + 30) / 60
	if delayMin < 0 {
		delayMin = 0
	}

	schedTime := baseDate.Add(time.Duration(schedDep) * time.Second)
	schedTimeStr := fmt.Sprintf("%02d:%02d", (schedDep/3600)%24, (schedDep%3600)/60)
	realTime := baseDate.Add(time.Duration(realDep) * time.Second)
	realTimeStr := fmt.Sprintf("%02d:%02d", (realDep/3600)%24, (realDep%3600)/60)

	return types.DepartureInfo{
		RID:                 cleanRID,
		RName:               name,
		RColor:              route.Color,
		RType:               route.RType,
		Picto:               route.Picto,
		Headsign:            headsign,
		DepSec:              realDep,
		Time:                realTime,
		TimeString:          realTimeStr,
		WaitMinutes:         waitMin,
		ScheduledDepSec:     schedDep,
		ScheduledTime:       &schedTime,
		ScheduledTimeString: schedTimeStr,
		DelaySec:            delaySec,
		DelayMinutes:        delayMin,
		IsDelayed:           isDelayed,
	}
}

func (e *Engine) GetNextDeparturesForStop(stopID string, targetTime time.Time) []types.DepartureInfo {
	targetTime = utils.InParis(targetTime)

	activeServices := e.GetActiveServices(targetTime)
	ad := e.GetActiveDelays(targetTime)
	currSec := uint32(targetTime.Hour()*3600 + targetTime.Minute()*60 + targetTime.Second())

	year, month, day := targetTime.Date()
	baseDate := time.Date(year, month, day, 0, 0, 0, 0, targetTime.Location())

	cleanSID := utils.IdRemoveIDFMPrefix(stopID)
	edges := e.GetEdgesForStop(stopID)

	var children []string
	if parent, ok := e.SearchParentsStopById(stopID); ok && len(parent.ChildrenStops) > 0 {
		children = parent.ChildrenStops
	} else if parent, ok := e.SearchParentsStopById(cleanSID); ok && len(parent.ChildrenStops) > 0 {
		children = parent.ChildrenStops
	} else if ch, ok := e.childrenByParent[cleanSID]; ok && len(ch) > 0 {
		children = ch
	}

	if len(children) > 0 {
		allResults := make([]types.DepartureInfo, 0)
		seenKey := make(map[string]bool)
		for _, childSID := range children {
			if utils.IdRemoveIDFMPrefix(childSID) == cleanSID || childSID == stopID {
				continue
			}
			for _, dep := range e.GetNextDeparturesForStop(childSID, targetTime) {
				key := fmt.Sprintf("%s-%s-%d", dep.RID, dep.Headsign, dep.DepSec)
				if !seenKey[key] {
					seenKey[key] = true
					allResults = append(allResults, dep)
				}
			}
		}

		routes := utils.IndexRoutesByID(e.Net.Routes)
		for i := range edges {
			edge := &edges[i]
			if edge.RID == "" {
				continue
			}
			rtSched, ok := e.FindNextDepartureRealtime(edge, currSec, activeServices, ad)
			if !ok {
				continue
			}
			route := routes[edge.RID]
			name := route.SName
			if name == "" {
				name = route.LName
			}
			cleanRID := utils.IdRemoveIDFMPrefix(route.RID)
			if cleanRID == "" {
				cleanRID = utils.IdRemoveIDFMPrefix(edge.RID)
			}
			dep := buildDepartureInfo(cleanRID, name, route, edge.Headsign, rtSched, currSec, baseDate)
			key := fmt.Sprintf("%s-%s-%d", dep.RID, dep.Headsign, dep.DepSec)
			if !seenKey[key] {
				seenKey[key] = true
				allResults = append(allResults, dep)
			}
		}

		sort.Slice(allResults, func(i, j int) bool {
			return allResults[i].DepSec < allResults[j].DepSec
		})
		return allResults
	}

	results := make([]types.DepartureInfo, 0)
	routes := utils.IndexRoutesByID(e.Net.Routes)

	for i := range edges {
		edge := &edges[i]
		if edge.RID == "" {
			continue
		}

		rtSched, ok := e.FindNextDepartureRealtime(edge, currSec, activeServices, ad)
		if !ok {
			continue
		}

		route := routes[edge.RID]
		name := route.SName
		if name == "" {
			name = route.LName
		}

		cleanRID := utils.IdRemoveIDFMPrefix(route.RID)
		if cleanRID == "" {
			cleanRID = utils.IdRemoveIDFMPrefix(edge.RID)
		}

		results = append(results, buildDepartureInfo(cleanRID, name, route, edge.Headsign, rtSched, currSec, baseDate))
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].DepSec < results[j].DepSec
	})

	return results
}

func (e *Engine) GetNextDeparturesForRoute(rid string, targetTime time.Time) map[string][]types.DepartureInfo {
	targetTime = utils.InParis(targetTime)

	activeServices := e.GetActiveServices(targetTime)
	ad := e.GetActiveDelays(targetTime)
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
		for i := range edges {
			edge := &edges[i]
			edgeCleanRID := utils.IdRemoveIDFMPrefix(edge.RID)
			if edge.RID != rid && edgeCleanRID != cleanRID && edgeCleanRID != cleanRouteRID {
				continue
			}

			rtSched, ok := e.FindNextDepartureRealtime(edge, currSec, activeServices, ad)
			if !ok {
				continue
			}

			cleanStopID := utils.IdRemoveIDFMPrefix(stopID)
			results[cleanStopID] = append(results[cleanStopID], buildDepartureInfo(cleanRouteRID, name, route, edge.Headsign, rtSched, currSec, baseDate))
		}
	}

	for stopID := range results {
		sort.Slice(results[stopID], func(i, j int) bool {
			return results[stopID][i].DepSec < results[stopID][j].DepSec
		})
	}

	return results
}
