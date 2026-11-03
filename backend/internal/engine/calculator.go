package engine

import (
	"fmt"
	"sync"

	"maps/backend/internal/types"
	"maps/backend/internal/utils"
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

	config.TargetTime = utils.InParis(config.TargetTime)
	targetTime := config.TargetTime
	activeServices := e.GetActiveServices(targetTime)
	ad := e.GetActiveDisruptions(targetTime)

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
		if ad.IsStopBlocked(s.SID) {
			continue
		}
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
		if ad.IsStopBlocked(s.SID) {
			continue
		}
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
		if okSt && !ad.IsStopBlocked(nearestStationDep.SID) {
			if utils.IsAccessible(nearestStationDep.WheelChair, config.WheelchairOnly) {
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
		if okSt && !ad.IsStopBlocked(nearestStationDep.SID) {
			if utils.IsAccessible(nearestStationDep.WheelChair, config.WheelchairOnly) {
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
			if ad.IsStopBlocked(s.SID) || !utils.IsAccessible(s.WheelChair, config.WheelchairOnly) {
				continue
			}
			d := utils.CoordsGetDistance(s.Long, s.Lat, lonFrom, latFrom)
			if d <= 7.5 {
				bp, _, ok := e.FindNearestBikeParking(s.Long, s.Lat, 0.5)
				if !ok {
					continue
				}

				pID := utils.IdRemoveIDFMPrefix(s.ParentStation)
				if pID == "" {
					pID = utils.IdRemoveIDFMPrefix(s.SID)
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
		if ad.IsStopBlocked(s.SID) || !utils.IsAccessible(s.WheelChair, config.WheelchairOnly) {
			continue
		}
		d := utils.CoordsGetDistance(s.Long, s.Lat, lonTo, latTo)
		walkSec := uint32((d / walkSpeed) * 3600)
		arrMap[s.SID] = candidateArrival{stop: s, walkSec: walkSec, distKm: d}
	}
	for _, s := range stationStops {
		if ad.IsStopBlocked(s.SID) || !utils.IsAccessible(s.WheelChair, config.WheelchairOnly) {
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
		toStops := e.SearchStopsByNeerCord(lonTo, latTo, 5)
		for _, s := range toStops {
			if !ad.IsStopBlocked(s.SID) {
				d := utils.CoordsGetDistance(s.Long, s.Lat, lonTo, latTo)
				walkSec := uint32((d / walkSpeed) * 3600)
				arrMap[s.SID] = candidateArrival{stop: s, walkSec: walkSec, distKm: d}
				break
			}
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
				SID          string
				TripIndex    uint32
				HasTransited bool
			}
			type routeNode struct {
				time     uint32
				cost     int
				prevEdge types.Edge
				prevKey  stateKey
				hasPrev  bool
			}
			type queueNode struct {
				SID          string
				TripIndex    uint32
				HasTransited bool
				Priority     int
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
				key := stateKey{SID: dep.stop.SID, TripIndex: 0, HasTransited: false}
				visited[key] = routeNode{time: initTime, cost: initCost, hasPrev: false}
				queue.Push(queueNode{SID: dep.stop.SID, TripIndex: 0, HasTransited: false, Priority: initCost}, initCost)
			}

			var destKey stateKey
			found := false

			transferPenalty := route.Factors.Transfers
			waitMultiplier := route.Factors.Wait
			walkMultiplier := route.Factors.Walk

			for queue.Len() > 0 {
				current, _ := queue.Pop()
				currKey := stateKey{SID: current.SID, TripIndex: current.TripIndex, HasTransited: current.HasTransited}

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

				if current.TripIndex > 0 {
					for _, edge := range e.Graph[current.SID] {
						if edge.RID == "" || ad.IsEdgeBlocked(&edge) {
							continue
						}
						if !utils.IsAccessible(edge.Wheelchair, route.Config.WheelchairOnly) {
							continue
						}
						if len(disallowedRoutes) > 0 && disallowedRoutes[edge.RID] {
							continue
						}

						sched, okTrip := utils.FindTripDeparture(edge.Schedules, current.TripIndex, currTime, activeServices)
						if !okTrip {
							continue
						}

						waitTime := int(sched.Dep - currTime)
						rideTime := int(sched.Arr - sched.Dep)
						edgeCost := int(float64(rideTime)*0.7) + waitTime
						if ad.IsEdgePerturbed(&edge) {
							edgeCost += int(float64(rideTime)*0.3) + 60
						}

						tentativeCost := currCost + edgeCost
						nextKey := stateKey{SID: edge.ToSID, TripIndex: current.TripIndex, HasTransited: true}

						best, exists := visited[nextKey]
						if !exists || tentativeCost < best.cost {
							edgeCopy := edge
							edgeCopy.TripIndex = current.TripIndex
							edgeCopy.DepTime = sched.Dep
							edgeCopy.ArrTime = sched.Arr
							edgeCopy.Weight = int(sched.Arr - sched.Dep)
							if int(current.TripIndex)-1 >= 0 && int(current.TripIndex)-1 < len(e.Net.Trips) {
								edgeCopy.Headsign = e.Net.Trips[current.TripIndex-1].Headsign
							}

							visited[nextKey] = routeNode{
								time:     sched.Arr,
								cost:     tentativeCost,
								prevEdge: edgeCopy,
								prevKey:  currKey,
								hasPrev:  true,
							}
							queue.Push(queueNode{
								SID:          edge.ToSID,
								TripIndex:    current.TripIndex,
								HasTransited: true,
								Priority:     tentativeCost,
							}, tentativeCost)
						}
					}

					alightKey := stateKey{SID: current.SID, TripIndex: 0, HasTransited: true}
					bestAlight, existsAlight := visited[alightKey]
					if !existsAlight || currCost < bestAlight.cost {
						visited[alightKey] = routeNode{
							time:     currTime,
							cost:     currCost,
							prevKey:  currKey,
							prevEdge: types.Edge{FromSID: current.SID, ToSID: current.SID, Weight: 0},
							hasPrev:  true,
						}
						queue.Push(queueNode{
							SID:          current.SID,
							TripIndex:    0,
							HasTransited: true,
							Priority:     currCost,
						}, currCost)
					}
				} else {
					if arr, isArr := arrMap[current.SID]; isArr {
						walkTime := arr.walkSec
						nextTime := currTime + walkTime
						walkCost := int(float64(walkTime) * walkMultiplier)
						tentativeCost := currCost + walkCost
						nextKey := stateKey{SID: destVirtualSID, TripIndex: 0, HasTransited: current.HasTransited}

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
							queue.Push(queueNode{
								SID:          destVirtualSID,
								TripIndex:    0,
								HasTransited: current.HasTransited,
								Priority:     tentativeCost,
							}, tentativeCost)
						}
					}

					for _, edge := range e.Graph[current.SID] {
						if !utils.IsAccessible(edge.Wheelchair, route.Config.WheelchairOnly) {
							continue
						}
						if len(disallowedRoutes) > 0 && disallowedRoutes[edge.RID] {
							continue
						}

						if edge.RID == "" {
							if ad.IsStopBlocked(current.SID) || ad.IsStopBlocked(edge.ToSID) {
								continue
							}
							nextTime := currTime + uint32(edge.Weight)
							edgeCost := int(float64(edge.Weight) * walkMultiplier)
							tentativeCost := currCost + edgeCost
							nextKey := stateKey{SID: edge.ToSID, TripIndex: 0, HasTransited: current.HasTransited}

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
									SID:          edge.ToSID,
									TripIndex:    0,
									HasTransited: current.HasTransited,
									Priority:     tentativeCost,
								}, tentativeCost)
							}
						} else {
							if ad.IsEdgeBlocked(&edge) || ad.IsStopBlocked(current.SID) {
								continue
							}

							earliestDep := currTime
							if current.HasTransited {
								earliestDep = currTime + 120
							}
							scheds := utils.FindNextDepartures(edge.Schedules, earliestDep, 3, 1800, activeServices)
							for _, sched := range scheds {
								nextTime := sched.Arr
								waitTime := int(sched.Dep - currTime)
								rideTime := int(sched.Arr - sched.Dep)
								edgeCost := rideTime + int(float64(waitTime)*waitMultiplier)
								if current.HasTransited {
									edgeCost += transferPenalty
								}
								if ad.IsEdgePerturbed(&edge) {
									edgeCost += int(float64(rideTime)*0.3) + 60
								}

								tentativeCost := currCost + edgeCost
								nextKey := stateKey{SID: edge.ToSID, TripIndex: sched.TripIndex, HasTransited: true}

								best, exists := visited[nextKey]
								if !exists || tentativeCost < best.cost {
									edgeCopy := edge
									edgeCopy.TripIndex = sched.TripIndex
									edgeCopy.DepTime = sched.Dep
									edgeCopy.ArrTime = sched.Arr
									edgeCopy.Weight = int(sched.Arr - sched.Dep)
									if sched.TripIndex > 0 && int(sched.TripIndex)-1 < len(e.Net.Trips) {
										edgeCopy.Headsign = e.Net.Trips[sched.TripIndex-1].Headsign
									}

									visited[nextKey] = routeNode{
										time:     nextTime,
										cost:     tentativeCost,
										prevEdge: edgeCopy,
										prevKey:  currKey,
										hasPrev:  true,
									}
									queue.Push(queueNode{
										SID:          edge.ToSID,
										TripIndex:    sched.TripIndex,
										HasTransited: true,
										Priority:     tentativeCost,
									}, tentativeCost)
								}
							}

						}
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
				if !(node.prevEdge.FromSID == node.prevEdge.ToSID && node.prevEdge.Weight == 0) {
					path = append(path, node.prevEdge)
				}
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
	cleanFromSID := utils.IdRemoveIDFMPrefix(fromSID)
	cleanToSID := utils.IdRemoveIDFMPrefix(toSID)
	if fromSID == toSID || cleanFromSID == cleanToSID {
		return []types.Edge{}, nil
	}

	if route.Config.TimeType == types.TimeTypeArrival {
		return e.CalculateRouteStopToStopWithArrivalTime(fromSID, toSID, route)
	}

	if route.Config.TimeType != types.TimeTypeDeparture {
		return []types.Edge{}, fmt.Errorf("not implemented")
	}

	if len(e.Graph[fromSID]) == 0 {
		if len(e.Graph["IDFM:"+cleanFromSID]) > 0 {
			fromSID = "IDFM:" + cleanFromSID
		} else if len(e.Graph[cleanFromSID]) > 0 {
			fromSID = cleanFromSID
		}
	}

	type stateKey struct {
		SID          string
		TripIndex    uint32
		HasTransited bool
	}

	type routeNode struct {
		time     uint32
		cost     int
		prevEdge types.Edge
		prevKey  stateKey
		hasPrev  bool
	}

	type queueNode struct {
		SID          string
		TripIndex    uint32
		HasTransited bool
		Priority     int
	}

	queue := utils.NewPriorityQueue[queueNode]()
	visited := make(map[stateKey]routeNode, 4096)

	route.Config.TargetTime = utils.InParis(route.Config.TargetTime)
	targetTime := route.Config.TargetTime
	activeServices := e.GetActiveServices(targetTime)
	ad := e.GetActiveDisruptions(targetTime)
	if ad.IsStopBlocked(fromSID) || ad.IsStopBlocked(toSID) {
		return nil, fmt.Errorf("station is blocked")
	}

	routes := utils.IndexRoutesByID(e.Net.Routes)

	startSec := uint32(targetTime.Hour()*3600 + targetTime.Minute()*60 + targetTime.Second())
	startKey := stateKey{SID: fromSID, TripIndex: 0, HasTransited: false}
	visited[startKey] = routeNode{
		time:    startSec,
		cost:    0,
		hasPrev: false,
	}

	queue.Push(queueNode{
		SID:          fromSID,
		TripIndex:    0,
		HasTransited: false,
		Priority:     0,
	}, 0)

	var destKey stateKey
	found := false

	var transferPenalty = route.Factors.Transfers
	var waitMultiplier = route.Factors.Wait
	var walkMultiplier = route.Factors.Walk

	for queue.Len() > 0 {
		current, _ := queue.Pop()
		currKey := stateKey{SID: current.SID, TripIndex: current.TripIndex, HasTransited: current.HasTransited}

		if current.SID == toSID || utils.IdRemoveIDFMPrefix(current.SID) == cleanToSID {
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

		if current.TripIndex > 0 {
			for _, edge := range e.Graph[current.SID] {
				if edge.RID == "" || ad.IsEdgeBlocked(&edge) {
					continue
				}
				if !utils.IsAccessible(edge.Wheelchair, route.Config.WheelchairOnly) {
					continue
				}
				r, okR := routes[edge.RID]
				if okR && len(route.Config.AllowedModes) > 0 && !utils.CheckAllowedRouteType(r.RType, route.Config.AllowedModes) {
					continue
				}

				sched, okTrip := utils.FindTripDeparture(edge.Schedules, current.TripIndex, currTime, activeServices)
				if !okTrip {
					continue
				}

				waitTime := int(sched.Dep - currTime)
				rideTime := int(sched.Arr - sched.Dep)
				edgeCost := int(float64(rideTime)*0.7) + waitTime
				if ad.IsEdgePerturbed(&edge) {
					edgeCost += int(float64(rideTime)*0.3) + 60
				}

				tentativeCost := currCost + edgeCost
				nextKey := stateKey{SID: edge.ToSID, TripIndex: current.TripIndex, HasTransited: true}

				best, exists := visited[nextKey]
				if !exists || tentativeCost < best.cost {
					edgeCopy := edge
					edgeCopy.TripIndex = current.TripIndex
					edgeCopy.DepTime = sched.Dep
					edgeCopy.ArrTime = sched.Arr
					edgeCopy.Weight = int(sched.Arr - sched.Dep)
					if int(current.TripIndex)-1 >= 0 && int(current.TripIndex)-1 < len(e.Net.Trips) {
						edgeCopy.Headsign = e.Net.Trips[current.TripIndex-1].Headsign
					}

					visited[nextKey] = routeNode{
						time:     sched.Arr,
						cost:     tentativeCost,
						prevEdge: edgeCopy,
						prevKey:  currKey,
						hasPrev:  true,
					}
					queue.Push(queueNode{
						SID:          edge.ToSID,
						TripIndex:    current.TripIndex,
						HasTransited: true,
						Priority:     tentativeCost,
					}, tentativeCost)
				}
			}

			alightKey := stateKey{SID: current.SID, TripIndex: 0, HasTransited: true}
			bestAlight, existsAlight := visited[alightKey]
			if !existsAlight || currCost < bestAlight.cost {
				visited[alightKey] = routeNode{
					time:     currTime,
					cost:     currCost,
					prevKey:  currKey,
					prevEdge: types.Edge{FromSID: current.SID, ToSID: current.SID, Weight: 0},
					hasPrev:  true,
				}
				queue.Push(queueNode{
					SID:          current.SID,
					TripIndex:    0,
					HasTransited: true,
					Priority:     currCost,
				}, currCost)
			}
		} else {
			for _, edge := range e.Graph[current.SID] {
				if !utils.IsAccessible(edge.Wheelchair, route.Config.WheelchairOnly) {
					continue
				}
				r, okR := routes[edge.RID]
				if okR && len(route.Config.AllowedModes) > 0 && !utils.CheckAllowedRouteType(r.RType, route.Config.AllowedModes) {
					continue
				}

				if edge.RID == "" {
					if ad.IsStopBlocked(current.SID) || ad.IsStopBlocked(edge.ToSID) {
						continue
					}
					nextTime := currTime + uint32(edge.Weight)
					edgeCost := int(float64(edge.Weight) * walkMultiplier)
					tentativeCost := currCost + edgeCost
					nextKey := stateKey{SID: edge.ToSID, TripIndex: 0, HasTransited: current.HasTransited}

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
							SID:          edge.ToSID,
							TripIndex:    0,
							HasTransited: current.HasTransited,
							Priority:     tentativeCost,
						}, tentativeCost)
					}
				} else {
					if ad.IsEdgeBlocked(&edge) || ad.IsStopBlocked(current.SID) {
						continue
					}

					earliestDep := currTime
					if current.HasTransited {
						earliestDep = currTime + 120
					}
					scheds := utils.FindNextDepartures(edge.Schedules, earliestDep, 3, 1800, activeServices)
					for _, sched := range scheds {
						nextTime := sched.Arr
						waitTime := int(sched.Dep - currTime)
						rideTime := int(sched.Arr - sched.Dep)
						edgeCost := rideTime + int(float64(waitTime)*waitMultiplier)
						if current.HasTransited {
							edgeCost += transferPenalty
						}
						if ad.IsEdgePerturbed(&edge) {
							edgeCost += int(float64(rideTime)*0.3) + 60
						}

						tentativeCost := currCost + edgeCost
						nextKey := stateKey{SID: edge.ToSID, TripIndex: sched.TripIndex, HasTransited: true}

						best, exists := visited[nextKey]
						if !exists || tentativeCost < best.cost {
							edgeCopy := edge
							edgeCopy.TripIndex = sched.TripIndex
							edgeCopy.DepTime = sched.Dep
							edgeCopy.ArrTime = sched.Arr
							edgeCopy.Weight = int(sched.Arr - sched.Dep)
							if sched.TripIndex > 0 && int(sched.TripIndex)-1 < len(e.Net.Trips) {
								edgeCopy.Headsign = e.Net.Trips[sched.TripIndex-1].Headsign
							}

							visited[nextKey] = routeNode{
								time:     nextTime,
								cost:     tentativeCost,
								prevEdge: edgeCopy,
								prevKey:  currKey,
								hasPrev:  true,
							}
							queue.Push(queueNode{
								SID:          edge.ToSID,
								TripIndex:    sched.TripIndex,
								HasTransited: true,
								Priority:     tentativeCost,
							}, tentativeCost)
						}
					}

				}
			}
		}
	}

	if !found {
		return nil, fmt.Errorf("no path found %s - %s", fromSID, toSID)
	}

	currKey := destKey
	var path []types.Edge

	for {
		node, ok := visited[currKey]
		if !ok || !node.hasPrev {
			break
		}
		if !(node.prevEdge.FromSID == node.prevEdge.ToSID && node.prevEdge.Weight == 0) {
			path = append(path, node.prevEdge)
		}
		currKey = node.prevKey
	}

	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return path, nil
}

func (e *Engine) CalculateRouteCordsToCordsWithArrivalTime(lonFrom, latFrom, lonTo, latTo float64, config types.CalculatorConfig) (map[string][]types.Edge, error) {
	config.TargetTime = utils.InParis(config.TargetTime)
	targetTime := config.TargetTime
	activeServices := e.GetActiveServices(targetTime)
	ad := e.GetActiveDisruptions(targetTime)

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
		if ad.IsStopBlocked(s.SID) {
			continue
		}
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
		if ad.IsStopBlocked(s.SID) {
			continue
		}
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
		if okSt && !ad.IsStopBlocked(nearestStationDep.SID) {
			if utils.IsAccessible(nearestStationDep.WheelChair, config.WheelchairOnly) {
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
		if okSt && !ad.IsStopBlocked(nearestStationDep.SID) {
			if utils.IsAccessible(nearestStationDep.WheelChair, config.WheelchairOnly) {
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
			if ad.IsStopBlocked(s.SID) || !utils.IsAccessible(s.WheelChair, config.WheelchairOnly) {
				continue
			}
			d := utils.CoordsGetDistance(s.Long, s.Lat, lonFrom, latFrom)
			if d <= 7.5 {
				bp, _, ok := e.FindNearestBikeParking(s.Long, s.Lat, 0.5)
				if !ok {
					continue
				}

				pID := utils.IdRemoveIDFMPrefix(s.ParentStation)
				if pID == "" {
					pID = utils.IdRemoveIDFMPrefix(s.SID)
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
		if ad.IsStopBlocked(s.SID) || !utils.IsAccessible(s.WheelChair, config.WheelchairOnly) {
			continue
		}
		d := utils.CoordsGetDistance(s.Long, s.Lat, lonTo, latTo)
		walkSec := uint32((d / walkSpeed) * 3600)
		arrMap[s.SID] = candidateArrival{stop: s, walkSec: walkSec, distKm: d}
	}
	for _, s := range stationStops {
		if ad.IsStopBlocked(s.SID) || !utils.IsAccessible(s.WheelChair, config.WheelchairOnly) {
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
		toStops := e.SearchStopsByNeerCord(lonTo, latTo, 5)
		for _, s := range toStops {
			if !ad.IsStopBlocked(s.SID) {
				d := utils.CoordsGetDistance(s.Long, s.Lat, lonTo, latTo)
				walkSec := uint32((d / walkSpeed) * 3600)
				arrMap[s.SID] = candidateArrival{stop: s, walkSec: walkSec, distKm: d}
				break
			}
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
				SID          string
				TripIndex    uint32
				HasTransited bool
			}
			type routeNode struct {
				time     uint32
				cost     int
				prevEdge types.Edge
				prevKey  stateKey
				hasPrev  bool
			}
			type queueNode struct {
				SID          string
				TripIndex    uint32
				HasTransited bool
				Priority     int
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
				key := stateKey{SID: arr.stop.SID, TripIndex: 0, HasTransited: false}
				visited[key] = routeNode{time: initTime, cost: initCost, hasPrev: false}
				queue.Push(queueNode{SID: arr.stop.SID, TripIndex: 0, HasTransited: false, Priority: initCost}, initCost)
			}

			var originKey stateKey
			found := false

			transferPenalty := route.Factors.Transfers
			waitMultiplier := route.Factors.Wait
			walkMultiplier := route.Factors.Walk

			for queue.Len() > 0 {
				current, _ := queue.Pop()
				currKey := stateKey{SID: current.SID, TripIndex: current.TripIndex, HasTransited: current.HasTransited}

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

				if current.TripIndex > 0 {
					for _, edge := range e.ReverseGraph[current.SID] {
						if edge.RID == "" || ad.IsEdgeBlocked(&edge) {
							continue
						}
						if !utils.IsAccessible(edge.Wheelchair, route.Config.WheelchairOnly) {
							continue
						}
						if len(disallowedRoutes) > 0 && disallowedRoutes[edge.RID] {
							continue
						}

						sched, okTrip := utils.FindTripArrival(edge.Schedules, current.TripIndex, currTime, activeServices)
						if !okTrip {
							continue
						}

						waitTime := int(currTime - sched.Arr)
						rideTime := int(sched.Arr - sched.Dep)
						edgeCost := int(float64(rideTime)*0.7) + waitTime
						if ad.IsEdgePerturbed(&edge) {
							edgeCost += int(float64(rideTime)*0.3) + 60
						}

						tentativeCost := currCost + edgeCost
						nextKey := stateKey{SID: edge.FromSID, TripIndex: current.TripIndex, HasTransited: true}

						best, exists := visited[nextKey]
						if !exists || tentativeCost < best.cost {
							edgeCopy := edge
							edgeCopy.TripIndex = current.TripIndex
							edgeCopy.DepTime = sched.Dep
							edgeCopy.ArrTime = sched.Arr
							edgeCopy.Weight = int(sched.Arr - sched.Dep)
							if int(current.TripIndex)-1 >= 0 && int(current.TripIndex)-1 < len(e.Net.Trips) {
								edgeCopy.Headsign = e.Net.Trips[current.TripIndex-1].Headsign
							}

							visited[nextKey] = routeNode{
								time:     sched.Dep,
								cost:     tentativeCost,
								prevEdge: edgeCopy,
								prevKey:  currKey,
								hasPrev:  true,
							}
							queue.Push(queueNode{
								SID:          edge.FromSID,
								TripIndex:    current.TripIndex,
								HasTransited: true,
								Priority:     tentativeCost,
							}, tentativeCost)
						}
					}

					boardKey := stateKey{SID: current.SID, TripIndex: 0, HasTransited: true}
					bestBoard, existsBoard := visited[boardKey]
					if !existsBoard || currCost < bestBoard.cost {
						visited[boardKey] = routeNode{
							time:     currTime,
							cost:     currCost,
							prevKey:  currKey,
							prevEdge: types.Edge{FromSID: current.SID, ToSID: current.SID, Weight: 0},
							hasPrev:  true,
						}
						queue.Push(queueNode{
							SID:          current.SID,
							TripIndex:    0,
							HasTransited: true,
							Priority:     currCost,
						}, currCost)
					}
				} else {
					if dep, isDep := depCandidateMap[current.SID]; isDep {
						if currTime >= dep.timeSec {
							originDepTime := currTime - dep.timeSec
							depCost := dep.costSec
							if dep.mode == "walk" {
								depCost = int(float64(dep.costSec) * walkMultiplier)
							}
							tentativeCost := currCost + depCost
							nextKey := stateKey{SID: originVirtualSID, TripIndex: 0, HasTransited: current.HasTransited}

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
								queue.Push(queueNode{
									SID:          originVirtualSID,
									TripIndex:    0,
									HasTransited: current.HasTransited,
									Priority:     tentativeCost,
								}, tentativeCost)
							}
						}
					}

					for _, edge := range e.ReverseGraph[current.SID] {
						if !utils.IsAccessible(edge.Wheelchair, route.Config.WheelchairOnly) {
							continue
						}
						if len(disallowedRoutes) > 0 && disallowedRoutes[edge.RID] {
							continue
						}

						if edge.RID == "" {
							if ad.IsStopBlocked(edge.FromSID) || ad.IsStopBlocked(edge.ToSID) {
								continue
							}
							if currTime < uint32(edge.Weight) {
								continue
							}
							prevTime := currTime - uint32(edge.Weight)
							edgeCost := int(float64(edge.Weight) * walkMultiplier)
							tentativeCost := currCost + edgeCost
							nextKey := stateKey{SID: edge.FromSID, TripIndex: 0, HasTransited: current.HasTransited}

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
									SID:          edge.FromSID,
									TripIndex:    0,
									HasTransited: current.HasTransited,
									Priority:     tentativeCost,
								}, tentativeCost)
							}
						} else {
							if ad.IsEdgeBlocked(&edge) || ad.IsStopBlocked(current.SID) {
								continue
							}

							latestArr := currTime
							if current.HasTransited {
								if currTime < 120 {
									continue
								}
								latestArr = currTime - 120
							}

							scheds := utils.FindPrevDepartures(edge.Schedules, latestArr, 3, 1800, activeServices)
							for _, sched := range scheds {
								prevTime := sched.Dep
								waitTime := int(currTime - sched.Arr)
								rideTime := int(sched.Arr - sched.Dep)
								edgeCost := rideTime + int(float64(waitTime)*waitMultiplier)
								if current.HasTransited {
									edgeCost += transferPenalty
								}
								if ad.IsEdgePerturbed(&edge) {
									edgeCost += int(float64(rideTime)*0.3) + 60
								}

								tentativeCost := currCost + edgeCost
								nextKey := stateKey{SID: edge.FromSID, TripIndex: sched.TripIndex, HasTransited: true}

								best, exists := visited[nextKey]
								if !exists || tentativeCost < best.cost {
									edgeCopy := edge
									edgeCopy.TripIndex = sched.TripIndex
									edgeCopy.DepTime = sched.Dep
									edgeCopy.ArrTime = sched.Arr
									edgeCopy.Weight = int(sched.Arr - sched.Dep)
									if sched.TripIndex > 0 && int(sched.TripIndex)-1 < len(e.Net.Trips) {
										edgeCopy.Headsign = e.Net.Trips[sched.TripIndex-1].Headsign
									}

									visited[nextKey] = routeNode{
										time:     prevTime,
										cost:     tentativeCost,
										prevEdge: edgeCopy,
										prevKey:  currKey,
										hasPrev:  true,
									}
									queue.Push(queueNode{
										SID:          edge.FromSID,
										TripIndex:    sched.TripIndex,
										HasTransited: true,
										Priority:     tentativeCost,
									}, tentativeCost)
								}
							}

						}
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
				if !(node.prevEdge.FromSID == node.prevEdge.ToSID && node.prevEdge.Weight == 0) {
					path = append(path, node.prevEdge)
				}
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
	cleanFromSID := utils.IdRemoveIDFMPrefix(fromSID)
	cleanToSID := utils.IdRemoveIDFMPrefix(toSID)
	if fromSID == toSID || cleanFromSID == cleanToSID {
		return []types.Edge{}, nil
	}

	if len(e.ReverseGraph[toSID]) == 0 {
		if len(e.ReverseGraph["IDFM:"+cleanToSID]) > 0 {
			toSID = "IDFM:" + cleanToSID
		} else if len(e.ReverseGraph[cleanToSID]) > 0 {
			toSID = cleanToSID
		}
	}

	type stateKey struct {
		SID          string
		TripIndex    uint32
		HasTransited bool
	}

	type routeNode struct {
		time     uint32
		cost     int
		prevEdge types.Edge
		prevKey  stateKey
		hasPrev  bool
	}

	type queueNode struct {
		SID          string
		TripIndex    uint32
		HasTransited bool
		Priority     int
	}

	queue := utils.NewPriorityQueue[queueNode]()
	visited := make(map[stateKey]routeNode, 4096)

	routes := utils.IndexRoutesByID(e.Net.Routes)

	route.Config.TargetTime = utils.InParis(route.Config.TargetTime)
	targetTime := route.Config.TargetTime
	activeServices := e.GetActiveServices(targetTime)
	ad := e.GetActiveDisruptions(targetTime)
	if ad.IsStopBlocked(fromSID) || ad.IsStopBlocked(toSID) {
		return nil, fmt.Errorf("station is blocked")
	}

	targetSec := uint32(targetTime.Hour()*3600 + targetTime.Minute()*60 + targetTime.Second())
	startKey := stateKey{SID: toSID, TripIndex: 0, HasTransited: false}
	visited[startKey] = routeNode{
		time:    targetSec,
		cost:    0,
		hasPrev: false,
	}

	queue.Push(queueNode{
		SID:          toSID,
		TripIndex:    0,
		HasTransited: false,
		Priority:     0,
	}, 0)

	var originKey stateKey
	found := false

	var transferPenalty = route.Factors.Transfers
	var waitMultiplier = route.Factors.Wait
	var walkMultiplier = route.Factors.Walk

	for queue.Len() > 0 {
		current, _ := queue.Pop()
		currKey := stateKey{SID: current.SID, TripIndex: current.TripIndex, HasTransited: current.HasTransited}

		if current.SID == fromSID || utils.IdRemoveIDFMPrefix(current.SID) == cleanFromSID {
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

		if current.TripIndex > 0 {
			for _, edge := range e.ReverseGraph[current.SID] {
				if edge.RID == "" || ad.IsEdgeBlocked(&edge) {
					continue
				}
				if !utils.IsAccessible(edge.Wheelchair, route.Config.WheelchairOnly) {
					continue
				}
				r, okR := routes[edge.RID]
				if okR && len(route.Config.AllowedModes) > 0 && !utils.CheckAllowedRouteType(r.RType, route.Config.AllowedModes) {
					continue
				}

				sched, okTrip := utils.FindTripArrival(edge.Schedules, current.TripIndex, currTime, activeServices)
				if !okTrip {
					continue
				}

				waitTime := int(currTime - sched.Arr)
				rideTime := int(sched.Arr - sched.Dep)
				edgeCost := int(float64(rideTime)*0.7) + waitTime
				if ad.IsEdgePerturbed(&edge) {
					edgeCost += int(float64(rideTime)*0.3) + 60
				}

				tentativeCost := currCost + edgeCost
				nextKey := stateKey{SID: edge.FromSID, TripIndex: current.TripIndex, HasTransited: true}

				best, exists := visited[nextKey]
				if !exists || tentativeCost < best.cost {
					edgeCopy := edge
					edgeCopy.TripIndex = current.TripIndex
					edgeCopy.DepTime = sched.Dep
					edgeCopy.ArrTime = sched.Arr
					edgeCopy.Weight = int(sched.Arr - sched.Dep)
					if int(current.TripIndex)-1 >= 0 && int(current.TripIndex)-1 < len(e.Net.Trips) {
						edgeCopy.Headsign = e.Net.Trips[current.TripIndex-1].Headsign
					}

					visited[nextKey] = routeNode{
						time:     sched.Dep,
						cost:     tentativeCost,
						prevEdge: edgeCopy,
						prevKey:  currKey,
						hasPrev:  true,
					}
					queue.Push(queueNode{
						SID:          edge.FromSID,
						TripIndex:    current.TripIndex,
						HasTransited: true,
						Priority:     tentativeCost,
					}, tentativeCost)
				}
			}

			boardKey := stateKey{SID: current.SID, TripIndex: 0, HasTransited: true}
			bestBoard, existsBoard := visited[boardKey]
			if !existsBoard || currCost < bestBoard.cost {
				visited[boardKey] = routeNode{
					time:     currTime,
					cost:     currCost,
					prevKey:  currKey,
					prevEdge: types.Edge{FromSID: current.SID, ToSID: current.SID, Weight: 0},
					hasPrev:  true,
				}
				queue.Push(queueNode{
					SID:          current.SID,
					TripIndex:    0,
					HasTransited: true,
					Priority:     currCost,
				}, currCost)
			}
		} else {
			for _, edge := range e.ReverseGraph[current.SID] {
				if !utils.IsAccessible(edge.Wheelchair, route.Config.WheelchairOnly) {
					continue
				}
				r, okR := routes[edge.RID]
				if okR && len(route.Config.AllowedModes) > 0 && !utils.CheckAllowedRouteType(r.RType, route.Config.AllowedModes) {
					continue
				}

				if edge.RID == "" {
					if ad.IsStopBlocked(edge.FromSID) || ad.IsStopBlocked(edge.ToSID) {
						continue
					}
					if currTime < uint32(edge.Weight) {
						continue
					}
					prevTime := currTime - uint32(edge.Weight)
					edgeCost := int(float64(edge.Weight) * walkMultiplier)
					tentativeCost := currCost + edgeCost
					nextKey := stateKey{SID: edge.FromSID, TripIndex: 0, HasTransited: current.HasTransited}

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
							SID:          edge.FromSID,
							TripIndex:    0,
							HasTransited: current.HasTransited,
							Priority:     tentativeCost,
						}, tentativeCost)
					}
				} else {
					if ad.IsEdgeBlocked(&edge) || ad.IsStopBlocked(current.SID) {
						continue
					}

					latestArr := currTime
					if current.HasTransited {
						if currTime < 120 {
							continue
						}
						latestArr = currTime - 120
					}

					scheds := utils.FindPrevDepartures(edge.Schedules, latestArr, 3, 1800, activeServices)
					for _, sched := range scheds {
						prevTime := sched.Dep
						waitTime := int(currTime - sched.Arr)
						rideTime := int(sched.Arr - sched.Dep)
						edgeCost := rideTime + int(float64(waitTime)*waitMultiplier)
						if current.HasTransited {
							edgeCost += transferPenalty
						}
						if ad.IsEdgePerturbed(&edge) {
							edgeCost += int(float64(rideTime)*0.3) + 60
						}

						tentativeCost := currCost + edgeCost
						nextKey := stateKey{SID: edge.FromSID, TripIndex: sched.TripIndex, HasTransited: true}

						best, exists := visited[nextKey]
						if !exists || tentativeCost < best.cost {
							edgeCopy := edge
							edgeCopy.TripIndex = sched.TripIndex
							edgeCopy.DepTime = sched.Dep
							edgeCopy.ArrTime = sched.Arr
							edgeCopy.Weight = int(sched.Arr - sched.Dep)
							if sched.TripIndex > 0 && int(sched.TripIndex)-1 < len(e.Net.Trips) {
								edgeCopy.Headsign = e.Net.Trips[sched.TripIndex-1].Headsign
							}

							visited[nextKey] = routeNode{
								time:     prevTime,
								cost:     tentativeCost,
								prevEdge: edgeCopy,
								prevKey:  currKey,
								hasPrev:  true,
							}
							queue.Push(queueNode{
								SID:          edge.FromSID,
								TripIndex:    sched.TripIndex,
								HasTransited: true,
								Priority:     tentativeCost,
							}, tentativeCost)
						}
					}

				}
			}
		}
	}

	if !found {
		return nil, fmt.Errorf("no path found %s - %s", fromSID, toSID)
	}

	currKey := originKey
	var path []types.Edge

	for {
		node, ok := visited[currKey]
		if !ok || !node.hasPrev {
			break
		}
		if !(node.prevEdge.FromSID == node.prevEdge.ToSID && node.prevEdge.Weight == 0) {
			path = append(path, node.prevEdge)
		}
		currKey = node.prevKey
	}

	return path, nil
}
