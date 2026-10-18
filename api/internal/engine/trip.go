package engine

import (
	"maps/api/internal/types"
	"maps/api/internal/utils"
	"time"
)

func (e *Engine) EdgesToParsedTrip(edges []types.Edge, departureTime time.Time) *types.ParsedTrip {
	if len(edges) == 0 {
		return nil
	}

	stops := utils.IndexStopsByID(e.Net.Stops)
	routes := utils.IndexRoutesByID(e.Net.Routes)

	trip := &types.ParsedTrip{
		FromSID:        edges[0].FromSID,
		ToSID:          edges[len(edges)-1].ToSID,
		DepartureTime:  departureTime,
		IsWheelchair:   1,
		IsBikesAllowed: 1,
		Legs:           make([]types.TripLeg, 0),
	}

	year, month, day := departureTime.Date()
	baseDate := time.Date(year, month, day, 0, 0, 0, 0, departureTime.Location())
	currentSec := uint32(departureTime.Hour()*3600 + departureTime.Minute()*60 + departureTime.Second())

	totalDuration := 0
	walkingDuration := 0
	transfersCount := 0

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
					transfersCount++

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

	if len(trip.Legs) > 0 {
		trip.DepartureTime = trip.Legs[0].DepartureTime
		trip.ArrivalTime = trip.Legs[len(trip.Legs)-1].ArrivalTime
		totalDuration = int(trip.ArrivalTime.Sub(trip.DepartureTime).Seconds())
	}

	trip.DurationSec = totalDuration
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
