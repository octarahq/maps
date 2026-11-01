package engine

import (
	"fmt"
	"maps/backend/internal/types"
	"maps/backend/internal/utils"
	"time"
)

func (e *Engine) getStopFromIndex(sid string, stops map[string]types.Stop) types.Stop {
	if s, ok := stops[sid]; ok && s.Name != "" {
		return s
	}
	clean := utils.IdRemoveIDFMPrefix(sid)
	if s, ok := stops[clean]; ok && s.Name != "" {
		return s
	}
	if s, ok := e.SearchStopById(sid); ok && s.Name != "" {
		return s
	}
	return types.Stop{SID: sid}
}

func (e *Engine) EdgesToParsedTrip(
	edges []types.Edge,
	lonFrom, latFrom, lonTo, latTo float64,
	config types.CalculatorConfig,
) *types.ParsedTrip {
	if len(edges) == 0 {
		return nil
	}

	stops := utils.IndexStopsByID(e.Net.Stops)
	routes := utils.IndexRoutesByID(e.Net.Routes)

	firstStop := e.getStopFromIndex(edges[0].FromSID, stops)
	lastStop := e.getStopFromIndex(edges[len(edges)-1].ToSID, stops)

	targetTime := config.TargetTime
	if targetTime.IsZero() {
		targetTime = time.Now()
	}

	year, month, day := targetTime.Date()
	baseDate := time.Date(year, month, day, 0, 0, 0, 0, targetTime.Location())

	if config.TimeType == types.TimeTypeArrival {
		return e.edgesToParsedTripArrival(edges, lonFrom, latFrom, lonTo, latTo, config, stops, routes, firstStop, lastStop, baseDate, targetTime)
	}

	return e.edgesToParsedTripDeparture(edges, lonFrom, latFrom, lonTo, latTo, config, stops, routes, firstStop, lastStop, baseDate, targetTime)
}

func (e *Engine) edgesToParsedTripDeparture(
	edges []types.Edge,
	lonFrom, latFrom, lonTo, latTo float64,
	config types.CalculatorConfig,
	stops map[string]types.Stop,
	routes map[string]types.Route,
	firstStop, lastStop types.Stop,
	baseDate, targetTime time.Time,
) *types.ParsedTrip {
	activeServices := e.GetActiveServices(targetTime)
	ad := e.GetActiveDisruptions(targetTime)
	startSec := uint32(targetTime.Hour()*3600 + targetTime.Minute()*60 + targetTime.Second())
	currentSec := startSec

	trip := &types.ParsedTrip{
		FromSID:        utils.IdRemoveIDFMPrefix(edges[0].FromSID),
		ToSID:          utils.IdRemoveIDFMPrefix(edges[len(edges)-1].ToSID),
		DepartureTime:  targetTime,
		IsWheelchair:   types.AccessibilityYes,
		IsBikesAllowed: types.AccessibilityYes,
		Legs:           make([]types.TripLeg, 0),
	}

	var bp *types.BikeParking
	var bpDistM float64
	walkingDuration := 0
	bikeDuration := 0

	hasOriginCoords := (lonFrom != 0 || latFrom != 0)
	if hasOriginCoords {
		distDep := utils.CoordsGetDistance(firstStop.Long, firstStop.Lat, lonFrom, latFrom)
		bikeTimeSec := uint32((distDep / 15.0) * 3600.0)
		walkTimeSec := uint32((distDep / 4.8) * 3600.0)

		isBike := false
		if config.WithBike && distDep > 1.0 {
			bpFound, distFound, okP := e.FindNearestBikeParking(firstStop.Long, firstStop.Lat, 0.5)
			if okP {
				isBike = true
				bp = bpFound
				bpDistM = distFound * 1000.0
			}
		}

		if isBike {
			legDep := baseDate.Add(time.Duration(currentSec) * time.Second)
			legArr := baseDate.Add(time.Duration(currentSec+bikeTimeSec) * time.Second)
			trip.Legs = append(trip.Legs, types.TripLeg{
				Type:          types.LegTypeBike,
				FromSID:       "",
				ToSID:         utils.IdRemoveIDFMPrefix(firstStop.SID),
				DepartureTime: legDep,
				ArrivalTime:   legArr,
				DurationSec:   int(bikeTimeSec),
				DistanceKm:    distDep,
				Wheelchair:    types.AccessibilityUnknown,
				BikesAllowed:  types.AccessibilityYes,
			})
			currentSec += bikeTimeSec
			bikeDuration += int(bikeTimeSec)
		} else if distDep > 0.05 {
			legDep := baseDate.Add(time.Duration(currentSec) * time.Second)
			legArr := baseDate.Add(time.Duration(currentSec+walkTimeSec) * time.Second)
			trip.Legs = append(trip.Legs, types.TripLeg{
				Type:          types.LegTypeWalk,
				FromSID:       "",
				ToSID:         utils.IdRemoveIDFMPrefix(firstStop.SID),
				DepartureTime: legDep,
				ArrivalTime:   legArr,
				DurationSec:   int(walkTimeSec),
				DistanceKm:    distDep,
				Wheelchair:    types.AccessibilityYes,
				BikesAllowed:  types.AccessibilityYes,
			})
			currentSec += walkTimeSec
			walkingDuration += int(walkTimeSec)
		}
	}

	var currentLeg *types.TripLeg
	for _, edge := range edges {
		legType := types.LegTypeTransit
		if edge.RID == "" {
			legType = types.LegTypeWalk
		}

		isNewLeg := currentLeg == nil ||
			currentLeg.Type != legType ||
			(legType == types.LegTypeTransit && currentLeg.Route != nil && currentLeg.Route.RID != utils.IdRemoveIDFMPrefix(edge.RID))

		if isNewLeg {
			if currentLeg != nil {
				currentLeg.ArrivalTime = baseDate.Add(time.Duration(currentSec) * time.Second)
				currentLeg.DurationSec = int(currentLeg.ArrivalTime.Sub(currentLeg.DepartureTime).Seconds())
				trip.Legs = append(trip.Legs, *currentLeg)

				if legType == types.LegTypeTransit {
					if currentLeg.Type == types.LegTypeTransit {
						transferSec := uint32(180)
						transferDep := baseDate.Add(time.Duration(currentSec) * time.Second)
						transferArr := baseDate.Add(time.Duration(currentSec+transferSec) * time.Second)

						fromStop := e.getStopFromIndex(edge.FromSID, stops)
						transferLeg := types.TripLeg{
							Type:          types.LegTypeWalk,
							FromSID:       utils.IdRemoveIDFMPrefix(edge.FromSID),
							ToSID:         utils.IdRemoveIDFMPrefix(edge.FromSID),
							FromStopName:  fromStop.Name,
							ToStopName:    fromStop.Name,
							DepartureTime: transferDep,
							ArrivalTime:   transferArr,
							DurationSec:   int(transferSec),
						}
						trip.Legs = append(trip.Legs, transferLeg)
						currentSec += transferSec
						walkingDuration += int(transferSec)
					}
				}
			}

			fromStop := e.getStopFromIndex(edge.FromSID, stops)
			leg := types.TripLeg{
				Type:              legType,
				FromSID:           utils.IdRemoveIDFMPrefix(edge.FromSID),
				ToSID:             utils.IdRemoveIDFMPrefix(edge.ToSID),
				FromStopName:      fromStop.Name,
				Headsign:          edge.Headsign,
				Wheelchair:        edge.Wheelchair,
				BikesAllowed:      edge.BikesAllowed,
				IntermediateStops: make([]types.IntermediateStop, 0),
			}

			if legType == types.LegTypeTransit {
				route := routes[edge.RID]

				leg.Route = &types.RouteSummary{
					RID:       utils.IdRemoveIDFMPrefix(route.RID),
					ShortName: route.SName,
					LongName:  route.LName,
					RouteType: route.RType,
					Color:     route.Color,
					TextColor: route.TColor,
				}

				tPos, hasPos := utils.IndexPositionInTrainsByTripKey(e.Net.PositionsInTrain)[fmt.Sprintf("%s-%s;%s", utils.IdRemoveIDFMPrefix(edge.FromSID), utils.IdRemoveIDFMPrefix(edge.ToSID), utils.IdRemoveIDFMPrefix(edge.RID))]
				if hasPos == true {
					leg.PositionInTrain = utils.TripParsePositionInTrain(tPos)
				}

				sched, ok := utils.FindNextDeparture(edge.Schedules, currentSec, activeServices)
				if ok {
					leg.DepartureTime = baseDate.Add(time.Duration(sched.Dep) * time.Second)
					currentSec = sched.Arr
				} else {
					leg.DepartureTime = baseDate.Add(time.Duration(currentSec) * time.Second)
					currentSec += uint32(edge.Weight)
				}
			} else {
				leg.DepartureTime = baseDate.Add(time.Duration(currentSec) * time.Second)
				currentSec += uint32(edge.Weight)
				walkingDuration += edge.Weight
			}

			currentLeg = &leg
		} else {
			intermediateStop := e.getStopFromIndex(edge.FromSID, stops)
			currentLeg.IntermediateStops = append(currentLeg.IntermediateStops, types.IntermediateStop{
				SID:           utils.IdRemoveIDFMPrefix(intermediateStop.SID),
				Name:          intermediateStop.Name,
				ArrivalTime:   baseDate.Add(time.Duration(currentSec) * time.Second),
				DepartureTime: baseDate.Add(time.Duration(currentSec) * time.Second),
				IsBlocked:     ad.IsStopBlocked(intermediateStop.SID),
			})
			currentLeg.ToSID = utils.IdRemoveIDFMPrefix(edge.ToSID)

			if edge.RID != "" {
				sched, ok := utils.FindNextDeparture(edge.Schedules, currentSec, activeServices)
				if ok {
					currentSec = sched.Arr
				} else {
					currentSec += uint32(edge.Weight)
				}
			} else {
				currentSec += uint32(edge.Weight)
				walkingDuration += edge.Weight
			}

			if edge.Wheelchair == 0 {
				currentLeg.Wheelchair = 0
			}
			if edge.BikesAllowed == 0 {
				currentLeg.BikesAllowed = 0
			}
		}

		toStop := e.getStopFromIndex(edge.ToSID, stops)
		currentLeg.ToStopName = toStop.Name

		if edge.Wheelchair == 0 {
			trip.IsWheelchair = 0
		}
		if edge.BikesAllowed == 0 {
			trip.IsBikesAllowed = 0
		}
	}

	if currentLeg != nil {
		currentLeg.ArrivalTime = baseDate.Add(time.Duration(currentSec) * time.Second)
		currentLeg.DurationSec = int(currentLeg.ArrivalTime.Sub(currentLeg.DepartureTime).Seconds())
		trip.Legs = append(trip.Legs, *currentLeg)
	}

	hasDestCoords := (lonTo != 0 || latTo != 0)
	if hasDestCoords {
		distArr := utils.CoordsGetDistance(lastStop.Long, lastStop.Lat, lonTo, latTo)
		if distArr > 0.05 {
			walkArrSec := uint32((distArr / 4.8) * 3600.0)
			legDep := baseDate.Add(time.Duration(currentSec) * time.Second)
			legArr := baseDate.Add(time.Duration(currentSec+walkArrSec) * time.Second)
			trip.Legs = append(trip.Legs, types.TripLeg{
				Type:          types.LegTypeWalk,
				FromSID:       utils.IdRemoveIDFMPrefix(lastStop.SID),
				ToSID:         "",
				DepartureTime: legDep,
				ArrivalTime:   legArr,
				DurationSec:   int(walkArrSec),
				DistanceKm:    distArr,
				Wheelchair:    types.AccessibilityYes,
				BikesAllowed:  types.AccessibilityYes,
			})
			currentSec += walkArrSec
			walkingDuration += int(walkArrSec)
		}
	}

	if len(trip.Legs) > 0 {
		trip.DepartureTime = trip.Legs[0].DepartureTime
		trip.ArrivalTime = trip.Legs[len(trip.Legs)-1].ArrivalTime
		trip.DurationSec = int(trip.ArrivalTime.Sub(trip.DepartureTime).Seconds())
	}

	trip.WalkingDuration = walkingDuration
	trip.BikeDuration = bikeDuration
	trip.BikeParking = bp
	trip.BikeParkingDistM = bpDistM

	transitLegs := 0
	for _, l := range trip.Legs {
		if l.Type == types.LegTypeTransit {
			transitLegs++
		}
	}
	if transitLegs > 1 {
		trip.TransfersCount = transitLegs - 1
	} else {
		trip.TransfersCount = 0
	}

	return trip
}

func (e *Engine) edgesToParsedTripArrival(
	edges []types.Edge,
	lonFrom, latFrom, lonTo, latTo float64,
	config types.CalculatorConfig,
	stops map[string]types.Stop,
	routes map[string]types.Route,
	firstStop, lastStop types.Stop,
	baseDate, targetTime time.Time,
) *types.ParsedTrip {
	activeServices := e.GetActiveServices(targetTime)
	ad := e.GetActiveDisruptions(targetTime)
	targetArrSec := uint32(targetTime.Hour()*3600 + targetTime.Minute()*60 + targetTime.Second())
	currSec := targetArrSec

	n := len(edges)
	edgeDep := make([]uint32, n)
	edgeArr := make([]uint32, n)

	var destWalkSec uint32
	var distArr float64
	hasDestCoords := (lonTo != 0 || latTo != 0)
	if hasDestCoords {
		distArr = utils.CoordsGetDistance(lastStop.Long, lastStop.Lat, lonTo, latTo)
		if distArr > 0.05 {
			destWalkSec = uint32((distArr / 4.8) * 3600.0)
			if currSec >= destWalkSec {
				currSec -= destWalkSec
			}
		}
	}

	for i := n - 1; i >= 0; i-- {
		edge := edges[i]
		if edge.RID == "" {
			edgeArr[i] = currSec
			w := uint32(edge.Weight)
			if currSec >= w {
				currSec -= w
			}
			edgeDep[i] = currSec
		} else {
			depTime, arrTime, ok := utils.FindPrevDepartureWithArr(edge.Schedules, currSec, activeServices)
			if ok {
				edgeArr[i] = arrTime
				edgeDep[i] = depTime
				currSec = depTime
			} else {
				edgeArr[i] = currSec
				w := uint32(edge.Weight)
				if currSec >= w {
					currSec -= w
				}
				edgeDep[i] = currSec
			}
		}

		if i > 0 && edges[i].RID != "" && edges[i-1].RID != "" && edges[i].RID != edges[i-1].RID {
			transferSec := uint32(180)
			if currSec >= transferSec {
				currSec -= transferSec
			}
		}
	}

	var originDepSec, originArrSec uint32
	var originDistKm float64
	var isBike bool
	var bp *types.BikeParking
	var bpDistM float64

	originArrSec = currSec
	hasOriginCoords := (lonFrom != 0 || latFrom != 0)
	if hasOriginCoords {
		originDistKm = utils.CoordsGetDistance(firstStop.Long, firstStop.Lat, lonFrom, latFrom)
		bikeTimeSec := uint32((originDistKm / 15.0) * 3600.0)
		walkTimeSec := uint32((originDistKm / 4.8) * 3600.0)

		if config.WithBike && originDistKm > 1.0 {
			bpFound, distFound, okP := e.FindNearestBikeParking(firstStop.Long, firstStop.Lat, 0.5)
			if okP {
				isBike = true
				bp = bpFound
				bpDistM = distFound * 1000.0
			}
		}

		if isBike {
			if currSec >= bikeTimeSec {
				originDepSec = currSec - bikeTimeSec
			}
		} else if originDistKm > 0.05 {
			if currSec >= walkTimeSec {
				originDepSec = currSec - walkTimeSec
			}
		} else {
			originDepSec = currSec
		}
	} else {
		originDepSec = currSec
	}

	walkingDuration := 0
	bikeDuration := 0

	trip := &types.ParsedTrip{
		FromSID:        utils.IdRemoveIDFMPrefix(edges[0].FromSID),
		ToSID:          utils.IdRemoveIDFMPrefix(edges[n-1].ToSID),
		DepartureTime:  baseDate.Add(time.Duration(originDepSec) * time.Second),
		IsWheelchair:   types.AccessibilityYes,
		IsBikesAllowed: types.AccessibilityYes,
		Legs:           make([]types.TripLeg, 0),
	}

	if hasOriginCoords {
		if isBike {
			bikeDur := int(originArrSec - originDepSec)
			trip.Legs = append(trip.Legs, types.TripLeg{
				Type:          types.LegTypeBike,
				FromSID:       "",
				ToSID:         utils.IdRemoveIDFMPrefix(firstStop.SID),
				DepartureTime: baseDate.Add(time.Duration(originDepSec) * time.Second),
				ArrivalTime:   baseDate.Add(time.Duration(originArrSec) * time.Second),
				DurationSec:   bikeDur,
				DistanceKm:    originDistKm,
				Wheelchair:    types.AccessibilityUnknown,
				BikesAllowed:  types.AccessibilityYes,
			})
			bikeDuration += bikeDur
		} else if originDistKm > 0.05 {
			walkDur := int(originArrSec - originDepSec)
			trip.Legs = append(trip.Legs, types.TripLeg{
				Type:          types.LegTypeWalk,
				FromSID:       "",
				ToSID:         utils.IdRemoveIDFMPrefix(firstStop.SID),
				DepartureTime: baseDate.Add(time.Duration(originDepSec) * time.Second),
				ArrivalTime:   baseDate.Add(time.Duration(originArrSec) * time.Second),
				DurationSec:   walkDur,
				DistanceKm:    originDistKm,
				Wheelchair:    types.AccessibilityYes,
				BikesAllowed:  types.AccessibilityYes,
			})
			walkingDuration += walkDur
		}
	}

	var currentLeg *types.TripLeg
	for i, edge := range edges {
		legType := types.LegTypeTransit
		if edge.RID == "" {
			legType = types.LegTypeWalk
		}

		isNewLeg := currentLeg == nil ||
			currentLeg.Type != legType ||
			(legType == types.LegTypeTransit && currentLeg.Route != nil && currentLeg.Route.RID != utils.IdRemoveIDFMPrefix(edge.RID))

		if isNewLeg {
			if currentLeg != nil {
				trip.Legs = append(trip.Legs, *currentLeg)

				if legType == types.LegTypeTransit && currentLeg.Type == types.LegTypeTransit {
					transferSec := int(edgeDep[i] - uint32(currentLeg.ArrivalTime.Sub(baseDate).Seconds()))
					if transferSec <= 0 {
						transferSec = 180
					}
					tDep := currentLeg.ArrivalTime
					tArr := baseDate.Add(time.Duration(edgeDep[i]) * time.Second)
					if tArr.Before(tDep) {
						tArr = tDep.Add(time.Duration(transferSec) * time.Second)
					}

					fromStop := e.getStopFromIndex(edge.FromSID, stops)
					transferLeg := types.TripLeg{
						Type:          types.LegTypeWalk,
						FromSID:       utils.IdRemoveIDFMPrefix(edge.FromSID),
						ToSID:         utils.IdRemoveIDFMPrefix(edge.FromSID),
						FromStopName:  fromStop.Name,
						ToStopName:    fromStop.Name,
						DepartureTime: tDep,
						ArrivalTime:   tArr,
						DurationSec:   int(tArr.Sub(tDep).Seconds()),
					}
					trip.Legs = append(trip.Legs, transferLeg)
					walkingDuration += transferLeg.DurationSec
				}
			}

			fromStop := e.getStopFromIndex(edge.FromSID, stops)
			leg := types.TripLeg{
				Type:              legType,
				FromSID:           utils.IdRemoveIDFMPrefix(edge.FromSID),
				ToSID:             utils.IdRemoveIDFMPrefix(edge.ToSID),
				FromStopName:      fromStop.Name,
				Headsign:          edge.Headsign,
				Wheelchair:        edge.Wheelchair,
				BikesAllowed:      edge.BikesAllowed,
				DepartureTime:     baseDate.Add(time.Duration(edgeDep[i]) * time.Second),
				ArrivalTime:       baseDate.Add(time.Duration(edgeArr[i]) * time.Second),
				DurationSec:       int(edgeArr[i] - edgeDep[i]),
				IntermediateStops: make([]types.IntermediateStop, 0),
			}

			if legType == types.LegTypeTransit {
				route := routes[edge.RID]
				leg.Route = &types.RouteSummary{
					RID:       utils.IdRemoveIDFMPrefix(route.RID),
					ShortName: route.SName,
					LongName:  route.LName,
					RouteType: route.RType,
					Color:     route.Color,
					TextColor: route.TColor,
				}
				tPos, hasPos := utils.IndexPositionInTrainsByTripKey(e.Net.PositionsInTrain)[fmt.Sprintf("%s-%s;%s", utils.IdRemoveIDFMPrefix(edge.FromSID), utils.IdRemoveIDFMPrefix(edge.ToSID), utils.IdRemoveIDFMPrefix(edge.RID))]
				if hasPos {
					leg.PositionInTrain = utils.TripParsePositionInTrain(tPos)
				}
			} else {
				walkingDuration += edge.Weight
			}

			currentLeg = &leg
		} else {
			intermediateStop := e.getStopFromIndex(edge.FromSID, stops)
			currentLeg.IntermediateStops = append(currentLeg.IntermediateStops, types.IntermediateStop{
				SID:           utils.IdRemoveIDFMPrefix(intermediateStop.SID),
				Name:          intermediateStop.Name,
				ArrivalTime:   baseDate.Add(time.Duration(edgeDep[i]) * time.Second),
				DepartureTime: baseDate.Add(time.Duration(edgeDep[i]) * time.Second),
				IsBlocked:     ad.IsStopBlocked(intermediateStop.SID),
			})
			currentLeg.ToSID = utils.IdRemoveIDFMPrefix(edge.ToSID)
			currentLeg.ArrivalTime = baseDate.Add(time.Duration(edgeArr[i]) * time.Second)
			currentLeg.DurationSec = int(currentLeg.ArrivalTime.Sub(currentLeg.DepartureTime).Seconds())

			if edge.RID == "" {
				walkingDuration += edge.Weight
			}

			if edge.Wheelchair == 0 {
				currentLeg.Wheelchair = 0
			}
			if edge.BikesAllowed == 0 {
				currentLeg.BikesAllowed = 0
			}
		}

		toStop := e.getStopFromIndex(edge.ToSID, stops)
		currentLeg.ToStopName = toStop.Name

		if edge.Wheelchair == 0 {
			trip.IsWheelchair = 0
		}
		if edge.BikesAllowed == 0 {
			trip.IsBikesAllowed = 0
		}
	}

	if currentLeg != nil {
		trip.Legs = append(trip.Legs, *currentLeg)
	}

	if hasDestCoords && distArr > 0.05 {
		lastArr := trip.Legs[len(trip.Legs)-1].ArrivalTime
		destArr := lastArr.Add(time.Duration(destWalkSec) * time.Second)
		trip.Legs = append(trip.Legs, types.TripLeg{
			Type:          types.LegTypeWalk,
			FromSID:       utils.IdRemoveIDFMPrefix(lastStop.SID),
			ToSID:         "",
			DepartureTime: lastArr,
			ArrivalTime:   destArr,
			DurationSec:   int(destWalkSec),
			DistanceKm:    distArr,
			Wheelchair:    types.AccessibilityYes,
			BikesAllowed:  types.AccessibilityYes,
		})
		walkingDuration += int(destWalkSec)
	}

	if len(trip.Legs) > 0 {
		trip.DepartureTime = trip.Legs[0].DepartureTime
		trip.ArrivalTime = trip.Legs[len(trip.Legs)-1].ArrivalTime
		trip.DurationSec = int(trip.ArrivalTime.Sub(trip.DepartureTime).Seconds())
	}

	trip.WalkingDuration = walkingDuration
	trip.BikeDuration = bikeDuration
	trip.BikeParking = bp
	trip.BikeParkingDistM = bpDistM

	transitLegs := 0
	for _, l := range trip.Legs {
		if l.Type == types.LegTypeTransit {
			transitLegs++
		}
	}
	if transitLegs > 1 {
		trip.TransfersCount = transitLegs - 1
	} else {
		trip.TransfersCount = 0
	}

	return trip
}
