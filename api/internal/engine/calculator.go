package engine

import (
	"fmt"
	"maps/api/internal/types"
	"maps/api/internal/utils"
)

func (e *Engine) CalculateRouteCordsToCords(lonFrom, latFrom, lonTo, latTo float64) ([]types.Edge, error) {
	fromStop := e.SearchStopsByNeerCord(lonFrom, latFrom, 1)[0]
	toStop := e.SearchStopsByNeerCord(lonTo, latTo, 1)[0]

	return e.CalculateRouteStopToStop(fromStop.SID, toStop.SID)
}

func (e *Engine) CalculateRouteStopToStop(fromSID, toSID string) ([]types.Edge, error) {
	distances := make(map[string]int)
	previous := make(map[string]types.Edge)

	type node struct {
		SID      string
		Priority int
	}
	queue := utils.NewPriorityQueue[node]()

	distances[fromSID] = 0
	queue.Push(node{
		SID:      fromSID,
		Priority: 0,
	}, 0)

	found := false

	for queue.Len() > 0 {
		current, _ := queue.Pop()

		if current.SID == toSID {
			found = true
			break
		}

		if current.Priority > distances[current.SID] {
			continue
		}

		for _, edge := range e.Graph[current.SID] {
			tentativeDist := distances[current.SID] + edge.Weight

			prevEdge := previous[current.SID]
			if current.SID != fromSID {
				if edge.RID != "" && edge.RID != prevEdge.RID {
					tentativeDist += 300
				}
				if edge.RID == "" && prevEdge.RID != "" {
					tentativeDist += 120
				}
			}

			currentBest, exists := distances[edge.ToSID]
			if !exists || tentativeDist < currentBest {
				distances[edge.ToSID] = tentativeDist
				previous[edge.ToSID] = edge
				queue.Push(node{
					SID:      edge.ToSID,
					Priority: tentativeDist,
				}, tentativeDist)
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
