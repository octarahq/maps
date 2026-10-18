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
	if fromSID == toSID {
		return []types.Edge{}, nil
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

	startSec := uint32(dTime.Hour()*3600 + dTime.Minute()*60 + dTime.Second())
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

	const transferPenalty = 600
	const waitMultiplier = 1.8
	const walkMultiplier = 1.5

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
					edgeCost = rideTime + waitTime
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
