package engine

import (
	"fmt"
	"maps/api/internal/types"
	"maps/api/internal/utils"
	"time"
)

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

	firstStop, okFirst := stops[edges[0].FromSID]
	if !okFirst {
		firstStop, _ = e.SearchStopById(edges[0].FromSID)
	}

	lastStop, okLast := stops[edges[len(edges)-1].ToSID]
	if !okLast {
		lastStop, _ = e.SearchStopById(edges[len(edges)-1].ToSID)
	}

	targetTime := config.TargetTime
	if targetTime.IsZero() {
		targetTime = time.Now()
	}

	year, month, day := targetTime.Date()
	baseDate := time.Date(year, month, day, 0, 0, 0, 0, targetTime.Location())
	startSec := uint32(targetTime.Hour()*3600 + targetTime.Minute()*60 + targetTime.Second())
	currentSec := startSec

	trip := &types.ParsedTrip{
		FromSID:        edges[0].FromSID,
		ToSID:          edges[len(edges)-1].ToSID,
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
				ToSID:         firstStop.SID,
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
				ToSID:         firstStop.SID,
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
			(legType == types.LegTypeTransit && currentLeg.Route != nil && currentLeg.Route.RID != edge.RID)

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

						fromStop := stops[edge.FromSID]
						transferLeg := types.TripLeg{
							Type:          types.LegTypeWalk,
							FromSID:       edge.FromSID,
							ToSID:         edge.FromSID,
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

			fromStop := stops[edge.FromSID]
			leg := types.TripLeg{
				Type:              legType,
				FromSID:           edge.FromSID,
				ToSID:             edge.ToSID,
				FromStopName:      fromStop.Name,
				Headsign:          edge.Headsign,
				Wheelchair:        edge.Wheelchair,
				BikesAllowed:      edge.BikesAllowed,
				IntermediateStops: make([]types.IntermediateStop, 0),
			}

			if legType == types.LegTypeTransit {
				route := routes[edge.RID]

				leg.Route = &types.RouteSummary{
					RID:       route.RID,
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

				sched, ok := utils.FindNextDeparture(edge.Schedules, currentSec)
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
			intermediateStop := stops[edge.FromSID]
			currentLeg.IntermediateStops = append(currentLeg.IntermediateStops, types.IntermediateStop{
				SID:           intermediateStop.SID,
				Name:          intermediateStop.Name,
				ArrivalTime:   baseDate.Add(time.Duration(currentSec) * time.Second),
				DepartureTime: baseDate.Add(time.Duration(currentSec) * time.Second),
			})
			currentLeg.ToSID = edge.ToSID

			if edge.RID != "" {
				sched, ok := utils.FindNextDeparture(edge.Schedules, currentSec)
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

		toStop := stops[edge.ToSID]
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
				FromSID:       lastStop.SID,
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
