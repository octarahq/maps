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

	targetTime := utils.InParis(config.TargetTime)
	config.TargetTime = targetTime

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
				FromLat:       latFrom,
				FromLon:       lonFrom,
				ToLat:         firstStop.Lat,
				ToLon:         firstStop.Long,
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
				FromLat:       latFrom,
				FromLon:       lonFrom,
				ToLat:         firstStop.Lat,
				ToLon:         firstStop.Long,
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
			(legType == types.LegTypeTransit && (currentLeg.TripIndex != edge.TripIndex || currentLeg.Route == nil || currentLeg.Route.RID != utils.IdRemoveIDFMPrefix(edge.RID)))

		if isNewLeg {
			if currentLeg != nil {
				currentLeg.ArrivalTime = baseDate.Add(time.Duration(currentSec) * time.Second)
				currentLeg.DurationSec = int(currentLeg.ArrivalTime.Sub(currentLeg.DepartureTime).Seconds())
				trip.Legs = append(trip.Legs, *currentLeg)

				if legType == types.LegTypeTransit {
					if currentLeg.Type == types.LegTypeTransit {
						transferSec := uint32(180)
						if edge.DepTime > currentSec {
							transferSec = edge.DepTime - currentSec
						}
						transferDep := baseDate.Add(time.Duration(currentSec) * time.Second)
						transferArr := baseDate.Add(time.Duration(currentSec+transferSec) * time.Second)

						fromStop := e.getStopFromIndex(edge.FromSID, stops)
						transferLeg := types.TripLeg{
							Type:          types.LegTypeWalk,
							FromSID:       utils.IdRemoveIDFMPrefix(edge.FromSID),
							ToSID:         utils.IdRemoveIDFMPrefix(edge.FromSID),
							FromStopName:  fromStop.Name,
							ToStopName:    fromStop.Name,
							FromLat:       fromStop.Lat,
							FromLon:       fromStop.Long,
							ToLat:         fromStop.Lat,
							ToLon:         fromStop.Long,
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
			toStop := e.getStopFromIndex(edge.ToSID, stops)
			leg := types.TripLeg{
				Type:              legType,
				TripIndex:         edge.TripIndex,
				FromSID:           utils.IdRemoveIDFMPrefix(edge.FromSID),
				ToSID:             utils.IdRemoveIDFMPrefix(edge.ToSID),
				FromStopName:      fromStop.Name,
				ToStopName:        toStop.Name,
				FromLat:           fromStop.Lat,
				FromLon:           fromStop.Long,
				ToLat:             toStop.Lat,
				ToLon:             toStop.Long,
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
					Picto:     route.Picto,
				}

				tPos, hasPos := utils.IndexPositionInTrainsByTripKey(e.Net.PositionsInTrain)[fmt.Sprintf("%s-%s;%s", utils.IdRemoveIDFMPrefix(edge.FromSID), utils.IdRemoveIDFMPrefix(edge.ToSID), utils.IdRemoveIDFMPrefix(edge.RID))]
				if hasPos == true {
					leg.PositionInTrain = utils.TripParsePositionInTrain(tPos)
				}

				if edge.DepTime > 0 && edge.ArrTime > 0 {
					leg.DepartureTime = baseDate.Add(time.Duration(edge.DepTime) * time.Second)
					currentSec = edge.ArrTime
				} else {
					sched, ok := utils.FindNextDeparture(edge.Schedules, currentSec, activeServices)
					if ok {
						leg.DepartureTime = baseDate.Add(time.Duration(sched.Dep) * time.Second)
						currentSec = sched.Arr
					} else {
						leg.DepartureTime = baseDate.Add(time.Duration(currentSec) * time.Second)
						currentSec += uint32(edge.Weight)
					}
				}
			} else {
				leg.DepartureTime = baseDate.Add(time.Duration(currentSec) * time.Second)
				currentSec += uint32(edge.Weight)
				walkingDuration += edge.Weight
			}

			currentLeg = &leg
		} else {
			intermediateStop := e.getStopFromIndex(edge.FromSID, stops)
			arrTime := currentSec
			if edge.DepTime > 0 {
				arrTime = edge.DepTime
			}
			currentLeg.IntermediateStops = append(currentLeg.IntermediateStops, types.IntermediateStop{
				SID:           utils.IdRemoveIDFMPrefix(intermediateStop.SID),
				Name:          intermediateStop.Name,
				ArrivalTime:   baseDate.Add(time.Duration(arrTime) * time.Second),
				DepartureTime: baseDate.Add(time.Duration(arrTime) * time.Second),
				IsBlocked:     ad.IsStopBlocked(intermediateStop.SID) || (edge.RID != "" && ad.IsStopBlockedOnLine(intermediateStop.SID, edge.RID)),
				Lat:           intermediateStop.Lat,
				Lon:           intermediateStop.Long,
			})
			currentLeg.ToSID = utils.IdRemoveIDFMPrefix(edge.ToSID)

			if edge.ArrTime > 0 {
				currentSec = edge.ArrTime
			} else if edge.RID != "" {
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
		currentLeg.ToLat = toStop.Lat
		currentLeg.ToLon = toStop.Long

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
				FromStopName:  lastStop.Name,
				ToStopName:    "",
				FromLat:       lastStop.Lat,
				FromLon:       lastStop.Long,
				ToLat:         latTo,
				ToLon:         lonTo,
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

	e.populatePositionsInTrain(trip)
	e.populateLegCoordinates(trip)

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
		} else if edge.DepTime > 0 && edge.ArrTime > 0 {
			edgeArr[i] = edge.ArrTime
			edgeDep[i] = edge.DepTime
			currSec = edge.DepTime
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

		if i > 0 && edges[i].RID != "" && edges[i-1].RID != "" && (edges[i].RID != edges[i-1].RID || edges[i].TripIndex != edges[i-1].TripIndex) {
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
				FromLat:       latFrom,
				FromLon:       lonFrom,
				ToLat:         firstStop.Lat,
				ToLon:         firstStop.Long,
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
				FromLat:       latFrom,
				FromLon:       lonFrom,
				ToLat:         firstStop.Lat,
				ToLon:         firstStop.Long,
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
			(legType == types.LegTypeTransit && (currentLeg.TripIndex != edge.TripIndex || currentLeg.Route == nil || currentLeg.Route.RID != utils.IdRemoveIDFMPrefix(edge.RID)))

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
						FromLat:       fromStop.Lat,
						FromLon:       fromStop.Long,
						ToLat:         fromStop.Lat,
						ToLon:         fromStop.Long,
						DepartureTime: tDep,
						ArrivalTime:   tArr,
						DurationSec:   int(tArr.Sub(tDep).Seconds()),
					}
					trip.Legs = append(trip.Legs, transferLeg)
					walkingDuration += transferLeg.DurationSec
				}
			}

			fromStop := e.getStopFromIndex(edge.FromSID, stops)
			toStop := e.getStopFromIndex(edge.ToSID, stops)
			leg := types.TripLeg{
				Type:              legType,
				TripIndex:         edge.TripIndex,
				FromSID:           utils.IdRemoveIDFMPrefix(edge.FromSID),
				ToSID:             utils.IdRemoveIDFMPrefix(edge.ToSID),
				FromStopName:      fromStop.Name,
				ToStopName:        toStop.Name,
				FromLat:           fromStop.Lat,
				FromLon:           fromStop.Long,
				ToLat:             toStop.Lat,
				ToLon:             toStop.Long,
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
					Picto:     route.Picto,
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
				IsBlocked:     ad.IsStopBlocked(intermediateStop.SID) || (edge.RID != "" && ad.IsStopBlockedOnLine(intermediateStop.SID, edge.RID)),
				Lat:           intermediateStop.Lat,
				Lon:           intermediateStop.Long,
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
		currentLeg.ToLat = toStop.Lat
		currentLeg.ToLon = toStop.Long

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
			FromStopName:  lastStop.Name,
			ToStopName:    "",
			FromLat:       lastStop.Lat,
			FromLon:       lastStop.Long,
			ToLat:         latTo,
			ToLon:         lonTo,
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

	e.populatePositionsInTrain(trip)
	e.populateLegCoordinates(trip)

	return trip
}

func (e *Engine) populatePositionsInTrain(trip *types.ParsedTrip) {
	if trip == nil || len(trip.Legs) == 0 || len(e.Net.PositionsInTrain) == 0 {
		return
	}
	pitMap := utils.IndexPositionInTrainsByTripKey(e.Net.PositionsInTrain)

	byFromAndLine := make(map[string]types.PositionInTrain, len(e.Net.PositionsInTrain))
	for _, pit := range e.Net.PositionsInTrain {
		k := fmt.Sprintf("%s;%s", utils.IdRemoveIDFMPrefix(pit.FromSID), utils.IdRemoveIDFMPrefix(pit.RID))
		if _, exists := byFromAndLine[k]; !exists {
			byFromAndLine[k] = pit
		}
	}

	for i := range trip.Legs {
		leg := &trip.Legs[i]
		if leg.Type != types.LegTypeTransit || leg.Route == nil {
			continue
		}

		var targetStops []string
		for j := i + 1; j < len(trip.Legs); j++ {
			next := &trip.Legs[j]
			if next.FromSID != "" {
				targetStops = append(targetStops, next.FromSID)
			}
			if next.ToSID != "" {
				targetStops = append(targetStops, next.ToSID)
			}
			if next.Type == types.LegTypeTransit {
				break
			}
		}

		found := false
		for _, target := range targetStops {
			key := fmt.Sprintf("%s-%s;%s", leg.ToSID, target, leg.Route.RID)
			if tPos, ok := pitMap[key]; ok {
				leg.PositionInTrain = utils.TripParsePositionInTrain(tPos)
				found = true
				break
			}
		}

		if !found {
			key := fmt.Sprintf("%s-%s;%s", leg.FromSID, leg.ToSID, leg.Route.RID)
			if tPos, ok := pitMap[key]; ok {
				leg.PositionInTrain = utils.TripParsePositionInTrain(tPos)
				found = true
			}
		}

		if !found {
			k := fmt.Sprintf("%s;%s", leg.ToSID, leg.Route.RID)
			if tPos, ok := byFromAndLine[k]; ok {
				leg.PositionInTrain = utils.TripParsePositionInTrain(tPos)
				found = true
			}
		}

		if !found {
			k := fmt.Sprintf("%s;%s", leg.FromSID, leg.Route.RID)
			if tPos, ok := byFromAndLine[k]; ok {
				leg.PositionInTrain = utils.TripParsePositionInTrain(tPos)
				found = true
			}
		}
	}
}

func sliceShape(pts [][2]float64, stopCoords [][2]float64) [][2]float64 {
	if len(pts) == 0 || len(stopCoords) == 0 {
		return nil
	}
	if len(stopCoords) == 1 {
		return pts
	}

	distSq := func(p, target [2]float64) float64 {
		dlat := p[0] - target[0]
		dlon := p[1] - target[1]
		return dlat*dlat + dlon*dlon
	}

	findClosestIdx := func(startIdx int, target [2]float64) int {
		bestIdx := startIdx
		bestDist := 1e18
		for i := startIdx; i < len(pts); i++ {
			d := distSq(pts[i], target)
			if d < bestDist {
				bestDist = d
				bestIdx = i
			}
		}
		return bestIdx
	}

	firstIdx := findClosestIdx(0, stopCoords[0])
	if len(stopCoords) >= 2 {
		bestCombined := 1e18
		for i := 0; i < len(pts); i++ {
			d0 := distSq(pts[i], stopCoords[0])
			if d0 < 0.00005 {
				for j := i; j < len(pts); j++ {
					d1 := distSq(pts[j], stopCoords[1])
					if d0+d1 < bestCombined {
						bestCombined = d0 + d1
						firstIdx = i
					}
				}
			}
		}
	}

	currIdx := firstIdx
	for s := 1; s < len(stopCoords); s++ {
		currIdx = findClosestIdx(currIdx, stopCoords[s])
	}
	lastIdx := currIdx

	if firstIdx > lastIdx {
		firstIdx, lastIdx = lastIdx, firstIdx
	}

	count := lastIdx - firstIdx + 1
	if count <= 0 {
		return nil
	}

	res := make([][2]float64, count)
	copy(res, pts[firstIdx:lastIdx+1])
	return res
}

func (e *Engine) findBestShapeForLeg(stopCoords [][2]float64, rType types.RouteType) (string, [][2]float64) {
	if len(stopCoords) < 2 {
		return "", nil
	}
	shapesList := e.shapesByType[rType]
	if len(shapesList) == 0 {
		return "", nil
	}

	distSq := func(p, target [2]float64) float64 {
		dlat := p[0] - target[0]
		dlon := p[1] - target[1]
		return dlat*dlat + dlon*dlon
	}

	startPt := stopCoords[0]
	endPt := stopCoords[len(stopCoords)-1]

	var bestID string
	var bestPts [][2]float64
	bestScore := 1e18

	for _, shp := range shapesList {
		if len(shp.Points) < 2 {
			continue
		}

		if startPt[0] < shp.MinLat-0.01 || startPt[0] > shp.MaxLat+0.01 ||
			startPt[1] < shp.MinLon-0.01 || startPt[1] > shp.MaxLon+0.01 ||
			endPt[0] < shp.MinLat-0.01 || endPt[0] > shp.MaxLat+0.01 ||
			endPt[1] < shp.MinLon-0.01 || endPt[1] > shp.MaxLon+0.01 {
			continue
		}

		pts := shp.Points
		bestI := -1
		minD0 := 1e18
		for i := 0; i < len(pts); i++ {
			d := distSq(pts[i], startPt)
			if d < minD0 {
				minD0 = d
				bestI = i
			}
		}
		if minD0 > 0.0001 {
			continue
		}

		bestJ := -1
		minD1 := 1e18
		for j := bestI; j < len(pts); j++ {
			d := distSq(pts[j], endPt)
			if d < minD1 {
				minD1 = d
				bestJ = j
			}
		}
		if minD1 > 0.0001 || bestJ <= bestI {
			continue
		}

		interScore := 0.0
		currK := bestI
		allFound := true
		for s := 1; s < len(stopCoords)-1; s++ {
			bestK := currK
			minDk := 1e18
			for k := currK; k <= bestJ; k++ {
				d := distSq(pts[k], stopCoords[s])
				if d < minDk {
					minDk = d
					bestK = k
				}
			}
			if minDk > 0.0002 {
				allFound = false
				break
			}
			interScore += minDk
			currK = bestK
		}
		if !allFound {
			continue
		}

		totalScore := minD0 + minD1 + interScore
		if totalScore < bestScore {
			bestScore = totalScore
			bestID = shp.ID
			bestPts = pts
		}
	}

	return bestID, bestPts
}

func (e *Engine) populateLegCoordinates(trip *types.ParsedTrip) {
	if trip == nil || len(trip.Legs) == 0 {
		return
	}

	for i := range trip.Legs {
		leg := &trip.Legs[i]
		if leg.Type == types.LegTypeWalk || leg.Type == types.LegTypeBike {
			coords := make([][2]float64, 0, 2)
			if leg.FromLat != 0 || leg.FromLon != 0 {
				coords = append(coords, [2]float64{leg.FromLat, leg.FromLon})
			}
			if leg.ToLat != 0 || leg.ToLon != 0 {
				coords = append(coords, [2]float64{leg.ToLat, leg.ToLon})
			}
			leg.Coordinates = coords
			continue
		}

		if leg.Type == types.LegTypeTransit {
			var stopCoords [][2]float64
			if leg.FromLat != 0 || leg.FromLon != 0 {
				stopCoords = append(stopCoords, [2]float64{leg.FromLat, leg.FromLon})
			}
			for _, is := range leg.IntermediateStops {
				if is.Lat != 0 || is.Lon != 0 {
					stopCoords = append(stopCoords, [2]float64{is.Lat, is.Lon})
				}
			}
			if leg.ToLat != 0 || leg.ToLon != 0 {
				stopCoords = append(stopCoords, [2]float64{leg.ToLat, leg.ToLon})
			}

			var pts [][2]float64
			hasTrip := leg.TripIndex > 0 && int(leg.TripIndex)-1 < len(e.Net.Trips)

			if hasTrip && len(e.shapesIndex) > 0 {
				t := e.Net.Trips[leg.TripIndex-1]
				if t.ShapeID != "" {
					pts = e.shapesIndex[t.ShapeID]
					if len(pts) == 0 {
						pts = e.shapesIndex[utils.IdRemoveIDFMPrefix(t.ShapeID)]
					}
				}
			}

			if len(pts) == 0 && len(stopCoords) >= 2 {
				matchedID, matchedPts := e.findBestShapeForLeg(stopCoords, leg.Route.RouteType)
				if len(matchedPts) > 0 {
					pts = matchedPts
					if hasTrip {
						e.Net.Trips[leg.TripIndex-1].ShapeID = matchedID
					}
				}
			}

			if len(pts) > 0 && len(stopCoords) >= 2 {
				sliced := sliceShape(pts, stopCoords)
				if len(sliced) >= 2 {
					sliced[0] = stopCoords[0]
					sliced[len(sliced)-1] = stopCoords[len(stopCoords)-1]
					leg.Coordinates = sliced
					continue
				}
			}

			leg.Coordinates = stopCoords
		}
	}
}
