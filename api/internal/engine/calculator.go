package engine

import (
	"fmt"
	"maps/api/internal/types"
	"maps/api/internal/utils"
	"time"
)

func (e *Engine) CalculateRouteCordsToCordsWithDepartureTime(lonFrom, latFrom, lonTo, latTo float64, dTime time.Time) ([]types.Edge, error) {
	fromStop := e.SearchStopsByNeerCord(lonFrom, latFrom, 1)[0]
	toStop := e.SearchStopsByNeerCord(lonTo, latTo, 1)[0]

	return e.CalculateRouteStopToStopWithDepartureTime(fromStop.SID, toStop.SID, dTime)
}

func (e *Engine) CalculateRouteStopToStopWithDepartureTime(fromSID, toSID string, dTime time.Time) ([]types.Edge, error) {
	previous := make(map[string]types.Edge)

	type node struct {
		SID      string
		Priority int
	}
	queue := utils.NewPriorityQueue[node]()

	startSec := uint32(dTime.Hour()*3600 + dTime.Minute()*60 + dTime.Second())

	distances := make(map[string]uint32)
	distances[fromSID] = startSec

	queue.Push(node{
		SID:      fromSID,
		Priority: int(startSec),
	}, int(startSec))

	found := false

	for queue.Len() > 0 {
		current, _ := queue.Pop()

		if current.SID == toSID {
			found = true
			break
		}

		if uint32(current.Priority) > distances[current.SID] {
			continue
		}

		currTime := distances[current.SID]

		for _, edge := range e.Graph[current.SID] {
			var arrivalTime uint32

			if edge.RID == "" {
				arrivalTime = currTime + uint32(edge.Weight)
			} else {
				readyTime := currTime
				prevEdge := previous[current.SID]
				if current.SID != fromSID && prevEdge.RID != "" && edge.RID != prevEdge.RID {
					readyTime += 180
				}

				sched, ok := utils.FindNextDeparture(edge.Schedules, readyTime)
				if !ok {
					continue
				}
				arrivalTime = sched.Arr
			}

			currentBest, exists := distances[edge.ToSID]
			if !exists || arrivalTime < currentBest {
				distances[edge.ToSID] = arrivalTime
				previous[edge.ToSID] = edge
				queue.Push(node{
					SID:      edge.ToSID,
					Priority: int(arrivalTime),
				}, int(arrivalTime))
			}
		}
	}

	if !found && fromSID != toSID {
		return nil, fmt.Errorf("no path found %s - %s", fromSID, toSID)
	}

	curr := toSID
	var path []types.Edge

	for curr != fromSID {
		edge, ok := previous[curr]
		if !ok {
			return nil, fmt.Errorf("no path found")
		}
		path = append(path, edge)
		curr = edge.FromSID
	}

	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return path, nil
}
