package engine

import (
	"fmt"
	"maps/api/internal/types"
	"maps/api/internal/utils"
	"sync"
)

func (e *Engine) CalculateRouteCordsToCords(lonFrom, latFrom, lonTo, latTo float64, config types.CalculatorConfig) (map[string][]types.Edge, error) {
	if config.TimeType == types.TimeTypeArrival {
		return e.CalculateRouteCordsToCordsWithArrivalTime(lonFrom, latFrom, lonTo, latTo, config)
	}
	return e.CalculateRouteCordsToCordsWithDepartureTime(lonFrom, latFrom, lonTo, latTo, config)
}

func (e *Engine) CalculateRouteCordsToCordsWithDepartureTime(lonFrom, latFrom, lonTo, latTo float64, config types.CalculatorConfig) (map[string][]types.Edge, error) {
	if config.TimeType == types.TimeTypeArrival {
		return e.CalculateRouteCordsToCordsWithArrivalTime(lonFrom, latFrom, lonTo, latTo, config)
	}
	var routesToCalculate = []types.RoutesToCalculateType{
		{
			Name:   "comfort",
			Config: config,
			Factors: struct {
				Walk      float64
				Wait      float64
				Transfers int
			}{
				Walk:      1.8,
				Wait:      1.3,
				Transfers: 1200,
			},
		},
		{
			Name:   "fastest",
			Config: config,
			Factors: struct {
				Walk      float64
				Wait      float64
				Transfers int
			}{
				Walk:      1.5,
				Wait:      1.1,
				Transfers: 300,
			},
		},
		{
			Name:   "least_walk",
			Config: config,
			Factors: struct {
				Walk      float64
				Wait      float64
				Transfers int
			}{
				Walk:      4.0,
				Wait:      1.1,
				Transfers: 450,
			},
		},
		{
			Name:   "nearest_station",
			Config: config,
			Factors: struct {
				Walk      float64
				Wait      float64
				Transfers int
			}{
				Walk:      1.8,
				Wait:      1.3,
				Transfers: 600,
			},
		},
	}

	type candidateDeparture struct {
		stop        types.Stop
		timeSec     uint32
		costSec     int
		mode        string
		distKm      float64
		bikeParking *types.BikeParking
	}

	type candidateArrival struct {
		stop    types.Stop
		walkSec uint32
		distKm  float64
	}

	walkSpeed := 4.8
	bikeSpeed := 15.0
	stationStops := e.StationStops()

	var walkDeps []candidateDeparture
	depStops := e.SearchBoardingStopsByCoord(lonFrom, latFrom, 1.2, 15)
	if len(depStops) == 0 {
		depStops = e.SearchBoardingStopsByCoord(lonFrom, latFrom, 2.5, 15)
	}
	if len(depStops) == 0 {
		depStops = e.SearchStopsByNeerCord(lonFrom, latFrom, 5)
	}
	depMap := make(map[string]candidateDeparture)
	for _, s := range depStops {
		d := utils.CoordsGetDistance(s.Long, s.Lat, lonFrom, latFrom)
		if d <= 1.2 {
			walkSec := uint32((d / walkSpeed) * 3600)
			depMap[s.SID] = candidateDeparture{
				stop:    s,
				timeSec: walkSec,
				costSec: int(walkSec),
				mode:    "walk",
				distKm:  d,
			}
		}
	}
	for _, s := range stationStops {
		d := utils.CoordsGetDistance(s.Long, s.Lat, lonFrom, latFrom)
		if d <= 1.2 {
			walkSec := uint32((d / walkSpeed) * 3600)
			if existing, ok := depMap[s.SID]; !ok || existing.distKm > d {
				depMap[s.SID] = candidateDeparture{
					stop:    s,
					timeSec: walkSec,
					costSec: int(walkSec),
					mode:    "walk",
					distKm:  d,
				}
			}
		}
	}
	for _, dep := range depMap {
		walkDeps = append(walkDeps, dep)
	}

	var nearestStationDeps []candidateDeparture
	if config.WithBike {
		nearestStationDep, bp, okSt := e.SearchNearestStationWithBikeParkingByCoord(lonFrom, latFrom, 7.5, 0.5)
		if okSt {
			if !utils.IsAccessible(config.WheelchairOnly) || nearestStationDep.WheelChair != types.AccessibilityNo {
				d := utils.CoordsGetDistance(nearestStationDep.Long, nearestStationDep.Lat, lonFrom, latFrom)
				bikeSec := uint32((d / bikeSpeed) * 3600)
				nearestStationDeps = append(nearestStationDeps, candidateDeparture{
					stop:        nearestStationDep,
					timeSec:     bikeSec,
					costSec:     int(bikeSec),
					mode:        "bike",
					distKm:      d,
					bikeParking: bp,
				})
			}
		}
	} else {
		nearestStationDep, okSt := e.SearchNearestStationByCoord(lonFrom, latFrom, 1.5)
		if okSt {
			if !utils.IsAccessible(config.WheelchairOnly) || nearestStationDep.WheelChair != types.AccessibilityNo {
				d := utils.CoordsGetDistance(nearestStationDep.Long, nearestStationDep.Lat, lonFrom, latFrom)
				walkSec := uint32((d / walkSpeed) * 3600)
				nearestStationDeps = append(nearestStationDeps, candidateDeparture{
					stop:    nearestStationDep,
					timeSec: walkSec,
					costSec: int(walkSec),
					mode:    "walk",
					distKm:  d,
				})
			}
		}
	}

	var allBikeDeps []candidateDeparture
	if config.WithBike {
		seenDepParents := make(map[string]bool)
		for _, s := range stationStops {
			if utils.IsAccessible(config.WheelchairOnly) && s.WheelChair == types.AccessibilityNo {
				continue
			}
			d := utils.CoordsGetDistance(s.Long, s.Lat, lonFrom, latFrom)
			if d <= 7.5 {
				bp, _, ok := e.FindNearestBikeParking(s.Long, s.Lat, 0.5)
				if !ok {
					continue
				}

				pID := s.ParentStation
				if pID == "" {
					pID = s.SID
				}
				if seenDepParents[pID] {
					continue
				}
				seenDepParents[pID] = true
				bikeSec := uint32((d / bikeSpeed) * 3600)
				allBikeDeps = append(allBikeDeps, candidateDeparture{
					stop:        s,
					timeSec:     bikeSec,
					costSec:     int(bikeSec),
					mode:        "bike",
					distKm:      d,
					bikeParking: bp,
				})
			}
		}
	}

	arrMap := make(map[string]candidateArrival)
	for _, s := range e.SearchArrivalStopsByCoord(lonTo, latTo, 0.8, 15) {
		if utils.IsAccessible(config.WheelchairOnly) && s.WheelChair == types.AccessibilityNo {
			continue
		}
		d := utils.CoordsGetDistance(s.Long, s.Lat, lonTo, latTo)
		walkSec := uint32((d / walkSpeed) * 3600)
		arrMap[s.SID] = candidateArrival{stop: s, walkSec: walkSec, distKm: d}
	}
	for _, s := range stationStops {
		if utils.IsAccessible(config.WheelchairOnly) && s.WheelChair == types.AccessibilityNo {
			continue
		}
		d := utils.CoordsGetDistance(s.Long, s.Lat, lonTo, latTo)
		if d <= 1.5 {
			walkSec := uint32((d / walkSpeed) * 3600)
			if existing, ok := arrMap[s.SID]; !ok || existing.distKm > d {
				arrMap[s.SID] = candidateArrival{stop: s, walkSec: walkSec, distKm: d}
			}
		}
	}
	if len(arrMap) == 0 {
		toStops := e.SearchStopsByNeerCord(lonTo, latTo, 1)
		if len(toStops) > 0 {
			d := utils.CoordsGetDistance(toStops[0].Long, toStops[0].Lat, lonTo, latTo)
			walkSec := uint32((d / walkSpeed) * 3600)
			arrMap[toStops[0].SID] = candidateArrival{stop: toStops[0], walkSec: walkSec, distKm: d}
		}
	}

	disallowedRoutes := make(map[string]bool)
	if len(config.AllowedModes) > 0 && len(config.AllowedModes) < 4 {
		for _, r := range e.Net.Routes {
			if !utils.CheckAllowedRouteType(r.RType, config.AllowedModes) {
				disallowedRoutes[r.RID] = true
			}
		}
	}

	res := make(map[string][]types.Edge)
	destVirtualSID := "VIRTUAL_DEST_NODE"
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, route := range routesToCalculate {
		wg.Add(1)
		go func(route types.RoutesToCalculateType) {
			defer wg.Done()

			var deps []candidateDeparture
			if config.WithBike {
				switch route.Name {
				case "nearest_station":
					if len(nearestStationDeps) > 0 {
						deps = nearestStationDeps
					} else if len(allBikeDeps) > 0 {
						deps = allBikeDeps
					} else {
						deps = walkDeps
					}
				default:
					deps = append(allBikeDeps, walkDeps...)
				}
			} else {
				if route.Name == "nearest_station" && len(nearestStationDeps) > 0 {
					deps = nearestStationDeps
				} else {
					deps = walkDeps
				}
			}

			if len(deps) == 0 || len(arrMap) == 0 {
				return
			}

			type stateKey struct {
				SID     string
				lastRID string
			}
			type routeNode struct {
				time     uint32
				cost     int
				prevEdge types.Edge
				prevKey  stateKey
				hasPrev  bool
			}
			type queueNode struct {
				SID      string
				lastRID  string
				Priority int
			}

			queue := utils.NewPriorityQueue[queueNode]()
			visited := make(map[stateKey]routeNode, 4096)

			startSec := uint32(route.Config.TargetTime.Hour()*3600 + route.Config.TargetTime.Minute()*60 + route.Config.TargetTime.Second())
			for _, dep := range deps {
				initTime := startSec + dep.timeSec
				initCost := dep.costSec
				if dep.mode == "walk" {
					initCost = int(float64(dep.costSec) * route.Factors.Walk)
				}
				key := stateKey{SID: dep.stop.SID, lastRID: ""}
				visited[key] = routeNode{time: initTime, cost: initCost, hasPrev: false}
				queue.Push(queueNode{SID: dep.stop.SID, lastRID: "", Priority: initCost}, initCost)
			}

			var destKey stateKey
			found := false

			transferPenalty := route.Factors.Transfers
			waitMultiplier := route.Factors.Wait
			walkMultiplier := route.Factors.Walk

			for queue.Len() > 0 {
				current, _ := queue.Pop()
				currKey := stateKey{SID: current.SID, lastRID: current.lastRID}

				if current.SID == destVirtualSID {
					found = true
					destKey = currKey
					break
				}

				currentState, ok := visited[currKey]
				if !ok || current.Priority > currentState.cost {
					continue
				}

				currTime := currentState.time
				currCost := currentState.cost
				currLastRID := current.lastRID

				if arr, isArr := arrMap[current.SID]; isArr {
					walkTime := arr.walkSec
					nextTime := currTime + walkTime
					walkCost := int(float64(walkTime) * walkMultiplier)
					tentativeCost := currCost + walkCost
					nextKey := stateKey{SID: destVirtualSID, lastRID: ""}

					best, exists := visited[nextKey]
					if !exists || tentativeCost < best.cost {
						visited[nextKey] = routeNode{
							time:    nextTime,
							cost:    tentativeCost,
							prevKey: currKey,
							prevEdge: types.Edge{
								FromSID: current.SID,
								ToSID:   destVirtualSID,
								Weight:  int(walkTime),
							},
							hasPrev: true,
						}
						queue.Push(queueNode{SID: destVirtualSID, lastRID: "", Priority: tentativeCost}, tentativeCost)
					}
				}

				for _, edge := range e.Graph[current.SID] {
					if utils.IsAccessible(route.Config.WheelchairOnly) && edge.Wheelchair != types.AccessibilityYes && edge.Wheelchair != types.AccessibilityUnknown {
						continue
					}

					if len(disallowedRoutes) > 0 && disallowedRoutes[edge.RID] {
						continue
					}

					var nextTime uint32
					var edgeCost int
					nextLastRID := currLastRID

					if edge.RID == "" {
						nextTime = currTime + uint32(edge.Weight)
						edgeCost = int(float64(edge.Weight) * walkMultiplier)
					} else {
						depTime, arrTime, okNext := utils.FindNextDepartureWithDep(edge.Schedules, currTime)
						if !okNext {
							continue
						}
						nextTime = arrTime
						waitTime := int(depTime - currTime)
						rideTime := int(arrTime - depTime)

						if currLastRID == edge.RID {
							edgeCost = int(float64(rideTime)*0.7) + waitTime
						} else {
							edgeCost = rideTime + int(float64(waitTime)*waitMultiplier)
							if currLastRID != "" {
								edgeCost += transferPenalty
							}
						}
						nextLastRID = edge.RID
					}

					tentativeCost := currCost + edgeCost
					nextKey := stateKey{SID: edge.ToSID, lastRID: nextLastRID}

					best, exists := visited[nextKey]
					if !exists || tentativeCost < best.cost {
						visited[nextKey] = routeNode{
							time:     nextTime,
							cost:     tentativeCost,
							prevEdge: edge,
							prevKey:  currKey,
							hasPrev:  true,
						}
						queue.Push(queueNode{
							SID:      edge.ToSID,
							lastRID:  nextLastRID,
							Priority: tentativeCost,
						}, tentativeCost)
					}
				}
			}

			if !found {
				return
			}

			currKey := destKey
			var path []types.Edge

			for {
				node, ok := visited[currKey]
				if !ok || !node.hasPrev {
					break
				}
				path = append(path, node.prevEdge)
				currKey = node.prevKey
			}

			for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
				path[i], path[j] = path[j], path[i]
			}

			if len(path) > 0 && path[len(path)-1].ToSID == destVirtualSID {
				path = path[:len(path)-1]
			}

			mu.Lock()
			res[route.Name] = path
			mu.Unlock()
		}(route)
	}
	wg.Wait()

	return res, nil

}

func (e *Engine) CalculateRouteStopToStop(fromSID, toSID string, route types.RoutesToCalculateType) ([]types.Edge, error) {
	if route.Config.TimeType == types.TimeTypeArrival {
		return e.CalculateRouteStopToStopWithArrivalTime(fromSID, toSID, route)
	}
	return e.CalculateRouteStopToStopWithDepartureTime(fromSID, toSID, route)
}

func (e *Engine) CalculateRouteStopToStopWithDepartureTime(fromSID, toSID string, route types.RoutesToCalculateType) ([]types.Edge, error) {
	if fromSID == toSID {
		return []types.Edge{}, nil
	}

	if route.Config.TimeType == types.TimeTypeArrival {
		return e.CalculateRouteStopToStopWithArrivalTime(fromSID, toSID, route)
	}

	if route.Config.TimeType != types.TimeTypeDeparture {
		return []types.Edge{}, fmt.Errorf("not implemented")
	}

	type stateKey struct {
		SID     string
		lastRID string
	}

	type nodeState struct {
		time uint32
		cost int
	}

	type queueNode struct {
		SID      string
		lastRID  string
		Priority int
	}

	queue := utils.NewPriorityQueue[queueNode]()
	previous := make(map[stateKey]types.Edge)
	previousKey := make(map[stateKey]stateKey)
	bestStates := make(map[stateKey]nodeState)

	routes := utils.IndexRoutesByID(e.Net.Routes)

	startSec := uint32(route.Config.TargetTime.Hour()*3600 + route.Config.TargetTime.Minute()*60 + route.Config.TargetTime.Second())
	startKey := stateKey{SID: fromSID, lastRID: ""}
	bestStates[startKey] = nodeState{
		time: startSec,
		cost: 0,
	}

	queue.Push(queueNode{
		SID:      fromSID,
		lastRID:  "",
		Priority: 0,
	}, 0)

	var destKey stateKey
	found := false

	var transferPenalty = route.Factors.Transfers
	var waitMultiplier = route.Factors.Wait
	var walkMultiplier = route.Factors.Walk

	for queue.Len() > 0 {
		current, _ := queue.Pop()
		currKey := stateKey{SID: current.SID, lastRID: current.lastRID}

		if current.SID == toSID {
			found = true
			destKey = currKey
			break
		}

		currentState := bestStates[currKey]
		if current.Priority > currentState.cost {
			continue
		}

		currTime := currentState.time
		currCost := currentState.cost
		currLastRID := current.lastRID

		for _, edge := range e.Graph[current.SID] {
			var nextTime uint32
			var edgeCost int
			nextLastRID := currLastRID

			if utils.IsAccessible(route.Config.WheelchairOnly) && edge.Wheelchair != types.AccessibilityYes && edge.Wheelchair != types.AccessibilityUnknown {
				continue
			}

			r, ok := routes[edge.RID]
			if ok && len(route.Config.AllowedModes) > 0 && !utils.CheckAllowedRouteType(r.RType, route.Config.AllowedModes) {
				continue
			}

			if edge.RID == "" {
				nextTime = currTime + uint32(edge.Weight)
				edgeCost = int(float64(edge.Weight) * walkMultiplier)
			} else {
				depTime, arrTime, ok := utils.FindNextDepartureWithDep(edge.Schedules, currTime)
				if !ok {
					continue
				}

				nextTime = arrTime
				waitTime := int(depTime - currTime)
				rideTime := int(arrTime - depTime)

				if currLastRID == edge.RID {
					edgeCost = int(float64(rideTime)*0.7) + waitTime
				} else {
					edgeCost = rideTime + int(float64(waitTime)*waitMultiplier)
					if currLastRID != "" {
						edgeCost += transferPenalty
					}
				}
				nextLastRID = edge.RID
			}

			tentativeCost := currCost + edgeCost
			nextKey := stateKey{SID: edge.ToSID, lastRID: nextLastRID}

			best, exists := bestStates[nextKey]
			if !exists || tentativeCost < best.cost {
				bestStates[nextKey] = nodeState{
					time: nextTime,
					cost: tentativeCost,
				}
				previous[nextKey] = edge
				previousKey[nextKey] = currKey
				queue.Push(queueNode{
					SID:      edge.ToSID,
					lastRID:  nextLastRID,
					Priority: tentativeCost,
				}, tentativeCost)
			}
		}
	}

	if !found {
		return nil, fmt.Errorf("no path found %s - %s", fromSID, toSID)
	}

	currKey := destKey
	var path []types.Edge

	for currKey.SID != fromSID {
		edge, ok := previous[currKey]
		if !ok {
			return nil, fmt.Errorf("no path found")
		}
		path = append(path, edge)
		currKey = previousKey[currKey]
	}

	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return path, nil
}

func (e *Engine) CalculateRouteCordsToCordsWithArrivalTime(lonFrom, latFrom, lonTo, latTo float64, config types.CalculatorConfig) (map[string][]types.Edge, error) {
	var routesToCalculate = []types.RoutesToCalculateType{
		{
			Name:   "comfort",
			Config: config,
			Factors: struct {
				Walk      float64
				Wait      float64
				Transfers int
			}{
				Walk:      1.8,
				Wait:      1.3,
				Transfers: 1200,
			},
		},
		{
			Name:   "fastest",
			Config: config,
			Factors: struct {
				Walk      float64
				Wait      float64
				Transfers int
			}{
				Walk:      1.5,
				Wait:      1.1,
				Transfers: 300,
			},
		},
		{
			Name:   "least_walk",
			Config: config,
			Factors: struct {
				Walk      float64
				Wait      float64
				Transfers int
			}{
				Walk:      4.0,
				Wait:      1.1,
				Transfers: 450,
			},
		},
		{
			Name:   "nearest_station",
			Config: config,
			Factors: struct {
				Walk      float64
				Wait      float64
				Transfers int
			}{
				Walk:      1.8,
				Wait:      1.3,
				Transfers: 600,
			},
		},
	}

	type candidateDeparture struct {
		stop        types.Stop
		timeSec     uint32
		costSec     int
		mode        string
		distKm      float64
		bikeParking *types.BikeParking
	}

	type candidateArrival struct {
		stop    types.Stop
		walkSec uint32
		distKm  float64
	}

	walkSpeed := 4.8
	bikeSpeed := 15.0
	stationStops := e.StationStops()

	var walkDeps []candidateDeparture
	depStops := e.SearchBoardingStopsByCoord(lonFrom, latFrom, 1.2, 15)
	if len(depStops) == 0 {
		depStops = e.SearchBoardingStopsByCoord(lonFrom, latFrom, 2.5, 15)
	}
	if len(depStops) == 0 {
		depStops = e.SearchStopsByNeerCord(lonFrom, latFrom, 5)
	}
	depMap := make(map[string]candidateDeparture)
	for _, s := range depStops {
		d := utils.CoordsGetDistance(s.Long, s.Lat, lonFrom, latFrom)
		if d <= 1.2 {
			walkSec := uint32((d / walkSpeed) * 3600)
			depMap[s.SID] = candidateDeparture{
				stop:    s,
				timeSec: walkSec,
				costSec: int(walkSec),
				mode:    "walk",
				distKm:  d,
			}
		}
	}
	for _, s := range stationStops {
		d := utils.CoordsGetDistance(s.Long, s.Lat, lonFrom, latFrom)
		if d <= 1.2 {
			walkSec := uint32((d / walkSpeed) * 3600)
			if existing, ok := depMap[s.SID]; !ok || existing.distKm > d {
				depMap[s.SID] = candidateDeparture{
					stop:    s,
					timeSec: walkSec,
					costSec: int(walkSec),
					mode:    "walk",
					distKm:  d,
				}
			}
		}
	}
	for _, dep := range depMap {
		walkDeps = append(walkDeps, dep)
	}

	var nearestStationDeps []candidateDeparture
	if config.WithBike {
		nearestStationDep, bp, okSt := e.SearchNearestStationWithBikeParkingByCoord(lonFrom, latFrom, 7.5, 0.5)
		if okSt {
			if !utils.IsAccessible(config.WheelchairOnly) || nearestStationDep.WheelChair != types.AccessibilityNo {
				d := utils.CoordsGetDistance(nearestStationDep.Long, nearestStationDep.Lat, lonFrom, latFrom)
				bikeSec := uint32((d / bikeSpeed) * 3600)
				nearestStationDeps = append(nearestStationDeps, candidateDeparture{
					stop:        nearestStationDep,
					timeSec:     bikeSec,
					costSec:     int(bikeSec),
					mode:        "bike",
					distKm:      d,
					bikeParking: bp,
				})
			}
		}
	} else {
		nearestStationDep, okSt := e.SearchNearestStationByCoord(lonFrom, latFrom, 1.5)
		if okSt {
			if !utils.IsAccessible(config.WheelchairOnly) || nearestStationDep.WheelChair != types.AccessibilityNo {
				d := utils.CoordsGetDistance(nearestStationDep.Long, nearestStationDep.Lat, lonFrom, latFrom)
				walkSec := uint32((d / walkSpeed) * 3600)
				nearestStationDeps = append(nearestStationDeps, candidateDeparture{
					stop:    nearestStationDep,
					timeSec: walkSec,
					costSec: int(walkSec),
					mode:    "walk",
					distKm:  d,
				})
			}
		}
	}

	var allBikeDeps []candidateDeparture
	if config.WithBike {
		seenDepParents := make(map[string]bool)
		for _, s := range stationStops {
			if utils.IsAccessible(config.WheelchairOnly) && s.WheelChair == types.AccessibilityNo {
				continue
			}
			d := utils.CoordsGetDistance(s.Long, s.Lat, lonFrom, latFrom)
			if d <= 7.5 {
				bp, _, ok := e.FindNearestBikeParking(s.Long, s.Lat, 0.5)
				if !ok {
					continue
				}

				pID := s.ParentStation
				if pID == "" {
					pID = s.SID
				}
				if seenDepParents[pID] {
					continue
				}
				seenDepParents[pID] = true
				bikeSec := uint32((d / bikeSpeed) * 3600)
				allBikeDeps = append(allBikeDeps, candidateDeparture{
					stop:        s,
					timeSec:     bikeSec,
					costSec:     int(bikeSec),
					mode:        "bike",
					distKm:      d,
					bikeParking: bp,
				})
			}
		}
	}

	arrMap := make(map[string]candidateArrival)
	for _, s := range e.SearchArrivalStopsByCoord(lonTo, latTo, 0.8, 15) {
		if utils.IsAccessible(config.WheelchairOnly) && s.WheelChair == types.AccessibilityNo {
			continue
		}
		d := utils.CoordsGetDistance(s.Long, s.Lat, lonTo, latTo)
		walkSec := uint32((d / walkSpeed) * 3600)
		arrMap[s.SID] = candidateArrival{stop: s, walkSec: walkSec, distKm: d}
	}
	for _, s := range stationStops {
		if utils.IsAccessible(config.WheelchairOnly) && s.WheelChair == types.AccessibilityNo {
			continue
		}
		d := utils.CoordsGetDistance(s.Long, s.Lat, lonTo, latTo)
		if d <= 1.5 {
			walkSec := uint32((d / walkSpeed) * 3600)
			if existing, ok := arrMap[s.SID]; !ok || existing.distKm > d {
				arrMap[s.SID] = candidateArrival{stop: s, walkSec: walkSec, distKm: d}
			}
		}
	}
	if len(arrMap) == 0 {
		toStops := e.SearchStopsByNeerCord(lonTo, latTo, 1)
		if len(toStops) > 0 {
			d := utils.CoordsGetDistance(toStops[0].Long, toStops[0].Lat, lonTo, latTo)
			walkSec := uint32((d / walkSpeed) * 3600)
			arrMap[toStops[0].SID] = candidateArrival{stop: toStops[0], walkSec: walkSec, distKm: d}
		}
	}

	disallowedRoutes := make(map[string]bool)
	if len(config.AllowedModes) > 0 && len(config.AllowedModes) < 4 {
		for _, r := range e.Net.Routes {
			if !utils.CheckAllowedRouteType(r.RType, config.AllowedModes) {
				disallowedRoutes[r.RID] = true
			}
		}
	}

	res := make(map[string][]types.Edge)
	originVirtualSID := "VIRTUAL_ORIGIN_NODE"
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, route := range routesToCalculate {
		wg.Add(1)
		go func(route types.RoutesToCalculateType) {
			defer wg.Done()

			var deps []candidateDeparture
			if config.WithBike {
				switch route.Name {
				case "nearest_station":
					if len(nearestStationDeps) > 0 {
						deps = nearestStationDeps
					} else if len(allBikeDeps) > 0 {
						deps = allBikeDeps
					} else {
						deps = walkDeps
					}
				default:
					deps = append(allBikeDeps, walkDeps...)
				}
			} else {
				if route.Name == "nearest_station" && len(nearestStationDeps) > 0 {
					deps = nearestStationDeps
				} else {
					deps = walkDeps
				}
			}

			if len(deps) == 0 || len(arrMap) == 0 {
				return
			}

			depCandidateMap := make(map[string]candidateDeparture, len(deps))
			for _, dep := range deps {
				if existing, ok := depCandidateMap[dep.stop.SID]; !ok || dep.costSec < existing.costSec {
					depCandidateMap[dep.stop.SID] = dep
				}
			}

			type stateKey struct {
				SID     string
				lastRID string
			}
			type routeNode struct {
				time     uint32
				cost     int
				prevEdge types.Edge
				prevKey  stateKey
				hasPrev  bool
			}
			type queueNode struct {
				SID      string
				lastRID  string
				Priority int
			}

			queue := utils.NewPriorityQueue[queueNode]()
			visited := make(map[stateKey]routeNode, 4096)

			targetSec := uint32(route.Config.TargetTime.Hour()*3600 + route.Config.TargetTime.Minute()*60 + route.Config.TargetTime.Second())
			for _, arr := range arrMap {
				if targetSec < arr.walkSec {
					continue
				}
				initTime := targetSec - arr.walkSec
				initCost := int(float64(arr.walkSec) * route.Factors.Walk)
				key := stateKey{SID: arr.stop.SID, lastRID: ""}
				visited[key] = routeNode{time: initTime, cost: initCost, hasPrev: false}
				queue.Push(queueNode{SID: arr.stop.SID, lastRID: "", Priority: initCost}, initCost)
			}

			var originKey stateKey
			found := false

			transferPenalty := route.Factors.Transfers
			waitMultiplier := route.Factors.Wait
			walkMultiplier := route.Factors.Walk

			for queue.Len() > 0 {
				current, _ := queue.Pop()
				currKey := stateKey{SID: current.SID, lastRID: current.lastRID}

				if current.SID == originVirtualSID {
					found = true
					originKey = currKey
					break
				}

				currentState, ok := visited[currKey]
				if !ok || current.Priority > currentState.cost {
					continue
				}

				currTime := currentState.time
				currCost := currentState.cost
				currLastRID := current.lastRID

				if dep, isDep := depCandidateMap[current.SID]; isDep {
					if currTime >= dep.timeSec {
						originDepTime := currTime - dep.timeSec
						depCost := dep.costSec
						if dep.mode == "walk" {
							depCost = int(float64(dep.costSec) * walkMultiplier)
						}
						tentativeCost := currCost + depCost
						nextKey := stateKey{SID: originVirtualSID, lastRID: ""}

						best, exists := visited[nextKey]
						if !exists || tentativeCost < best.cost {
							visited[nextKey] = routeNode{
								time:    originDepTime,
								cost:    tentativeCost,
								prevKey: currKey,
								prevEdge: types.Edge{
									FromSID: originVirtualSID,
									ToSID:   current.SID,
									Weight:  int(dep.timeSec),
								},
								hasPrev: true,
							}
							queue.Push(queueNode{SID: originVirtualSID, lastRID: "", Priority: tentativeCost}, tentativeCost)
						}
					}
				}

				for _, edge := range e.ReverseGraph[current.SID] {
					if utils.IsAccessible(route.Config.WheelchairOnly) && edge.Wheelchair != types.AccessibilityYes && edge.Wheelchair != types.AccessibilityUnknown {
						continue
					}

					if len(disallowedRoutes) > 0 && disallowedRoutes[edge.RID] {
						continue
					}

					var prevTime uint32
					var edgeCost int
					nextLastRID := currLastRID

					if edge.RID == "" {
						if currTime < uint32(edge.Weight) {
							continue
						}
						prevTime = currTime - uint32(edge.Weight)
						edgeCost = int(float64(edge.Weight) * walkMultiplier)
					} else {
						depTime, arrTime, okPrev := utils.FindPrevDepartureWithArr(edge.Schedules, currTime)
						if !okPrev {
							continue
						}
						prevTime = depTime
						waitTime := int(currTime - arrTime)
						rideTime := int(arrTime - depTime)

						if currLastRID == edge.RID {
							edgeCost = int(float64(rideTime)*0.7) + waitTime
						} else {
							edgeCost = rideTime + int(float64(waitTime)*waitMultiplier)
							if currLastRID != "" {
								edgeCost += transferPenalty
							}
						}
						nextLastRID = edge.RID
					}

					tentativeCost := currCost + edgeCost
					nextKey := stateKey{SID: edge.FromSID, lastRID: nextLastRID}

					best, exists := visited[nextKey]
					if !exists || tentativeCost < best.cost {
						visited[nextKey] = routeNode{
							time:     prevTime,
							cost:     tentativeCost,
							prevEdge: edge,
							prevKey:  currKey,
							hasPrev:  true,
						}
						queue.Push(queueNode{
							SID:      edge.FromSID,
							lastRID:  nextLastRID,
							Priority: tentativeCost,
						}, tentativeCost)
					}
				}
			}

			if !found {
				return
			}

			currKey := visited[originKey].prevKey
			var path []types.Edge

			for {
				node, ok := visited[currKey]
				if !ok || !node.hasPrev {
					break
				}
				path = append(path, node.prevEdge)
				currKey = node.prevKey
			}

			mu.Lock()
			res[route.Name] = path
			mu.Unlock()
		}(route)
	}
	wg.Wait()

	return res, nil
}

func (e *Engine) CalculateRouteStopToStopWithArrivalTime(fromSID, toSID string, route types.RoutesToCalculateType) ([]types.Edge, error) {
	if fromSID == toSID {
		return []types.Edge{}, nil
	}

	type stateKey struct {
		SID     string
		lastRID string
	}

	type nodeState struct {
		time     uint32
		cost     int
		prevEdge types.Edge
		prevKey  stateKey
		hasPrev  bool
	}

	type queueNode struct {
		SID      string
		lastRID  string
		Priority int
	}

	queue := utils.NewPriorityQueue[queueNode]()
	bestStates := make(map[stateKey]nodeState)

	routes := utils.IndexRoutesByID(e.Net.Routes)

	targetSec := uint32(route.Config.TargetTime.Hour()*3600 + route.Config.TargetTime.Minute()*60 + route.Config.TargetTime.Second())
	startKey := stateKey{SID: toSID, lastRID: ""}
	bestStates[startKey] = nodeState{
		time:    targetSec,
		cost:    0,
		hasPrev: false,
	}

	queue.Push(queueNode{
		SID:      toSID,
		lastRID:  "",
		Priority: 0,
	}, 0)

	var originKey stateKey
	found := false

	var transferPenalty = route.Factors.Transfers
	var waitMultiplier = route.Factors.Wait
	var walkMultiplier = route.Factors.Walk

	for queue.Len() > 0 {
		current, _ := queue.Pop()
		currKey := stateKey{SID: current.SID, lastRID: current.lastRID}

		if current.SID == fromSID {
			found = true
			originKey = currKey
			break
		}

		currentState := bestStates[currKey]
		if current.Priority > currentState.cost {
			continue
		}

		currTime := currentState.time
		currCost := currentState.cost
		currLastRID := current.lastRID

		for _, edge := range e.ReverseGraph[current.SID] {
			var prevTime uint32
			var edgeCost int
			nextLastRID := currLastRID

			if utils.IsAccessible(route.Config.WheelchairOnly) && edge.Wheelchair != types.AccessibilityYes && edge.Wheelchair != types.AccessibilityUnknown {
				continue
			}

			r, ok := routes[edge.RID]
			if ok && len(route.Config.AllowedModes) > 0 && !utils.CheckAllowedRouteType(r.RType, route.Config.AllowedModes) {
				continue
			}

			if edge.RID == "" {
				if currTime < uint32(edge.Weight) {
					continue
				}
				prevTime = currTime - uint32(edge.Weight)
				edgeCost = int(float64(edge.Weight) * walkMultiplier)
			} else {
				depTime, arrTime, okNext := utils.FindPrevDepartureWithArr(edge.Schedules, currTime)
				if !okNext {
					continue
				}

				prevTime = depTime
				waitTime := int(currTime - arrTime)
				rideTime := int(arrTime - depTime)

				if currLastRID == edge.RID {
					edgeCost = int(float64(rideTime)*0.7) + waitTime
				} else {
					edgeCost = rideTime + int(float64(waitTime)*waitMultiplier)
					if currLastRID != "" {
						edgeCost += transferPenalty
					}
				}
				nextLastRID = edge.RID
			}

			tentativeCost := currCost + edgeCost
			nextKey := stateKey{SID: edge.FromSID, lastRID: nextLastRID}

			best, exists := bestStates[nextKey]
			if !exists || tentativeCost < best.cost {
				bestStates[nextKey] = nodeState{
					time:     prevTime,
					cost:     tentativeCost,
					prevEdge: edge,
					prevKey:  currKey,
					hasPrev:  true,
				}
				queue.Push(queueNode{
					SID:      edge.FromSID,
					lastRID:  nextLastRID,
					Priority: tentativeCost,
				}, tentativeCost)
			}
		}
	}

	if !found {
		return nil, fmt.Errorf("no path found %s - %s", fromSID, toSID)
	}

	currKey := originKey
	var path []types.Edge

	for {
		node, ok := bestStates[currKey]
		if !ok || !node.hasPrev {
			break
		}
		path = append(path, node.prevEdge)
		currKey = node.prevKey
	}

	return path, nil
}
