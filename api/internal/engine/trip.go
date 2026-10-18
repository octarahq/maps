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

	currentTime := departureTime
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
				currentLeg.ArrivalTime = currentTime
				currentLeg.DurationSec = int(currentLeg.ArrivalTime.Sub(currentLeg.DepartureTime).Seconds())
				trip.Legs = append(trip.Legs, *currentLeg)

				if legType == types.LegTypeTransit {
					transfersCount++

					if currentLeg.Type == types.LegTypeTransit {
						transferDuration := 180 * time.Second
						fromStop := stops[edge.FromSID]
						transferLeg := types.TripLeg{
							Type:          types.LegTypeWalk,
							FromSID:       edge.FromSID,
							ToSID:         edge.FromSID,
							FromStopName:  fromStop.Name,
							ToStopName:    fromStop.Name,
							DepartureTime: currentTime,
							ArrivalTime:   currentTime.Add(transferDuration),
							DurationSec:   int(transferDuration.Seconds()),
						}
						trip.Legs = append(trip.Legs, transferLeg)
						currentTime = currentTime.Add(transferDuration)
						walkingDuration += int(transferDuration.Seconds())
						totalDuration += int(transferDuration.Seconds())
					}
				}
			}

			fromStop := stops[edge.FromSID]
			leg := types.TripLeg{
				Type:              legType,
				FromSID:           edge.FromSID,
				ToSID:             edge.ToSID,
				FromStopName:      fromStop.Name,
				DepartureTime:     currentTime,
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
			}

			currentLeg = &leg
		} else {
			intermediateStop := stops[edge.FromSID]
			currentLeg.IntermediateStops = append(currentLeg.IntermediateStops, types.IntermediateStop{
				SID:           intermediateStop.SID,
				Name:          intermediateStop.Name,
				ArrivalTime:   currentTime,
				DepartureTime: currentTime,
			})
			currentLeg.ToSID = edge.ToSID

			if edge.Wheelchair == 0 {
				currentLeg.Wheelchair = 0
			}
			if edge.BikesAllowed == 0 {
				currentLeg.BikesAllowed = 0
			}
		}

		toStop := stops[edge.ToSID]
		currentLeg.ToStopName = toStop.Name

		edgeDuration := time.Duration(edge.Weight) * time.Second
		currentTime = currentTime.Add(edgeDuration)
		totalDuration += edge.Weight

		if legType == types.LegTypeWalk {
			walkingDuration += edge.Weight
		}

		if edge.Wheelchair == 0 {
			trip.IsWheelchair = 0
		}
		if edge.BikesAllowed == 0 {
			trip.IsBikesAllowed = 0
		}
	}

	if currentLeg != nil {
		currentLeg.ArrivalTime = currentTime
		currentLeg.DurationSec = int(currentLeg.ArrivalTime.Sub(currentLeg.DepartureTime).Seconds())
		trip.Legs = append(trip.Legs, *currentLeg)
	}

	trip.ArrivalTime = currentTime
	trip.DurationSec = totalDuration
	trip.WalkingDuration = walkingDuration
	trip.TransfersCount = transfersCount

	return trip
}
