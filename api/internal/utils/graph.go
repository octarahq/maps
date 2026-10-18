package utils

import (
	"cmp"
	"fmt"
	"slices"

	"maps/api/internal/types"
)

func BuildGraph(net *types.Network) types.GraphType {
	graph := make(types.GraphType)
	trips := IndexTripsByID(net.Trips)

	stopsTime := net.StopTimes
	savedStep := make(map[string]bool)

	type edgeKey struct {
		from string
		to   string
		rid  string
	}
	edgeIndices := make(map[edgeKey]int)

	for i := 1; i < len(stopsTime); i++ {
		prev := stopsTime[i-1]
		curr := stopsTime[i]

		if prev.TID == curr.TID {
			trip := trips[curr.TID]
			rid := trip.RID

			diff := int(curr.ATime) - int(prev.DTime)
			if diff <= 0 {
				diff = 30
			}

			key := edgeKey{from: prev.SId, to: curr.SId, rid: rid}
			idx, exists := edgeIndices[key]
			if !exists {
				edge := types.Edge{
					FromSID:      prev.SId,
					ToSID:        curr.SId,
					RID:          rid,
					Headsign:     trip.Headsign,
					Wheelchair:   trip.WheelChair,
					BikesAllowed: trip.BikesAllowed,
					Weight:       diff,
					Schedules: []types.DepartureTime{
						{Dep: uint32(prev.DTime), Arr: uint32(curr.ATime)},
					},
				}
				graph[prev.SId] = append(graph[prev.SId], edge)
				edgeIndices[key] = len(graph[prev.SId]) - 1
			} else {
				pEdge := &graph[prev.SId][idx]
				if pEdge.Headsign == "" && trip.Headsign != "" {
					pEdge.Headsign = trip.Headsign
				}
				if pEdge.Wheelchair == types.AccessibilityUnknown && trip.WheelChair != types.AccessibilityUnknown {
					pEdge.Wheelchair = trip.WheelChair
				}
				if pEdge.BikesAllowed == types.AccessibilityUnknown && trip.BikesAllowed != types.AccessibilityUnknown {
					pEdge.BikesAllowed = trip.BikesAllowed
				}
				pEdge.Schedules = append(pEdge.Schedules, types.DepartureTime{
					Dep: uint32(prev.DTime),
					Arr: uint32(curr.ATime),
				})
			}
		}
	}

	for fromSID := range graph {
		edges := graph[fromSID]
		for j := range edges {
			if len(edges[j].Schedules) > 1 {
				slices.SortFunc(edges[j].Schedules, func(a, b types.DepartureTime) int {
					return cmp.Compare(a.Dep, b.Dep)
				})
			}
		}
	}

	for _, tr := range net.Transfers {
		if tr.FromSID == tr.ToSID {
			continue
		}
		w := int(tr.MinTTime)
		if w <= 0 {
			w = 180
		}
		key := fmt.Sprintf("%s-%s@walk", tr.FromSID, tr.ToSID)
		if !savedStep[key] {
			graph[tr.FromSID] = append(graph[tr.FromSID], types.Edge{
				FromSID:      tr.FromSID,
				ToSID:        tr.ToSID,
				Weight:       w,
				RID:          "",
				Wheelchair:   types.AccessibilityYes,
				BikesAllowed: types.AccessibilityYes,
			})
			savedStep[key] = true
		}
	}

	platformsByParent := make(map[string][]string)
	for _, s := range net.Stops {
		if s.ParentStation != "" {
			platformsByParent[s.ParentStation] = append(platformsByParent[s.ParentStation], s.SID)
		}
	}

	for _, platforms := range platformsByParent {
		if len(platforms) < 2 {
			continue
		}
		for _, from := range platforms {
			for _, to := range platforms {
				if from != to {
					key := fmt.Sprintf("%s-%s@walk", from, to)
					if !savedStep[key] {
						graph[from] = append(graph[from], types.Edge{
							FromSID:      from,
							ToSID:        to,
							Weight:       180,
							RID:          "",
							Wheelchair:   types.AccessibilityYes,
							BikesAllowed: types.AccessibilityYes,
						})
						savedStep[key] = true
					}
				}
			}
		}
	}

	for _, s := range net.Stops {
		if s.ParentStation != "" {
			keyParentToQuai := fmt.Sprintf("%s-%s@zdc", s.ParentStation, s.SID)
			if !savedStep[keyParentToQuai] {
				graph[s.ParentStation] = append(graph[s.ParentStation], types.Edge{
					FromSID:      s.ParentStation,
					ToSID:        s.SID,
					Weight:       180,
					RID:          "",
					Wheelchair:   types.AccessibilityYes,
					BikesAllowed: types.AccessibilityYes,
				})
				savedStep[keyParentToQuai] = true
			}

			keyQuaiToParent := fmt.Sprintf("%s-%s@zdc", s.SID, s.ParentStation)
			if !savedStep[keyQuaiToParent] {
				graph[s.SID] = append(graph[s.SID], types.Edge{
					FromSID:      s.SID,
					ToSID:        s.ParentStation,
					Weight:       180,
					RID:          "",
					Wheelchair:   types.AccessibilityYes,
					BikesAllowed: types.AccessibilityYes,
				})
				savedStep[keyQuaiToParent] = true
			}
		}
	}

	return graph
}
