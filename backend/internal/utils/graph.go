package utils

import (
	"cmp"
	"slices"

	"maps/backend/internal/types"
)

func internString(m map[string]string, s string) string {
	if s == "" {
		return ""
	}
	if v, ok := m[s]; ok {
		return v
	}
	m[s] = s
	return s
}

func BuildGraph(net *types.Network, calIndex ...*CalendarIndex) types.GraphType {
	var ci *CalendarIndex
	if len(calIndex) > 0 {
		ci = calIndex[0]
	}

	intern := make(map[string]string, len(net.Stops)+len(net.Routes)+4096)
	graph := make(types.GraphType, len(net.Stops))
	trips := IndexTripsByID(net.Trips)
	tripIndexMap := make(map[string]uint32, len(net.Trips)*2)
	for i, t := range net.Trips {
		idx := uint32(i + 1)
		tripIndexMap[t.TID] = idx
		clean := IdRemoveIDFMPrefix(t.TID)
		if clean != t.TID {
			tripIndexMap[clean] = idx
		}
	}

	stopsTime := net.StopTimes
	type walkKey struct {
		from string
		to   string
		kind uint8
	}
	savedStep := make(map[walkKey]bool)

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
			rid := internString(intern, trip.RID)
			fromSID := internString(intern, prev.SId)
			toSID := internString(intern, curr.SId)
			headsign := internString(intern, trip.Headsign)
			tripIndex := tripIndexMap[curr.TID]

			var serviceID uint16
			if ci != nil && trip.ServiceID != "" {
				serviceID = ci.GetOrRegisterService(trip.ServiceID)
			}

			diff := int(curr.ATime) - int(prev.DTime)
			if diff <= 0 {
				diff = 30
			}

			key := edgeKey{from: fromSID, to: toSID, rid: rid}
			idx, exists := edgeIndices[key]
			if !exists {
				edge := types.Edge{
					FromSID:      fromSID,
					ToSID:        toSID,
					RID:          rid,
					Headsign:     headsign,
					Wheelchair:   trip.WheelChair,
					BikesAllowed: trip.BikesAllowed,
					Weight:       diff,
					Schedules: []types.DepartureTime{
						{Dep: uint32(prev.DTime), Arr: uint32(curr.ATime), ServiceID: serviceID, TripIndex: tripIndex},
					},
				}
				graph[fromSID] = append(graph[fromSID], edge)
				edgeIndices[key] = len(graph[fromSID]) - 1
			} else {
				pEdge := &graph[fromSID][idx]
				if pEdge.Headsign == "" && headsign != "" {
					pEdge.Headsign = headsign
				}
				if pEdge.Wheelchair == types.AccessibilityUnknown && trip.WheelChair != types.AccessibilityUnknown {
					pEdge.Wheelchair = trip.WheelChair
				}
				if pEdge.BikesAllowed == types.AccessibilityUnknown && trip.BikesAllowed != types.AccessibilityUnknown {
					pEdge.BikesAllowed = trip.BikesAllowed
				}
				pEdge.Schedules = append(pEdge.Schedules, types.DepartureTime{
					Dep:       uint32(prev.DTime),
					Arr:       uint32(curr.ATime),
					ServiceID: serviceID,
					TripIndex: tripIndex,
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
			edges[j].Schedules = slices.Clip(edges[j].Schedules)
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
		fromSID := internString(intern, tr.FromSID)
		toSID := internString(intern, tr.ToSID)
		key := walkKey{from: fromSID, to: toSID, kind: 1}
		if !savedStep[key] {
			graph[fromSID] = append(graph[fromSID], types.Edge{
				FromSID:      fromSID,
				ToSID:        toSID,
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
			ps := internString(intern, s.ParentStation)
			sid := internString(intern, s.SID)
			platformsByParent[ps] = append(platformsByParent[ps], sid)
		}
	}

	for _, platforms := range platformsByParent {
		if len(platforms) < 2 {
			continue
		}
		for _, from := range platforms {
			for _, to := range platforms {
				if from != to {
					key := walkKey{from: from, to: to, kind: 1}
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
			ps := internString(intern, s.ParentStation)
			sid := internString(intern, s.SID)
			keyParentToQuai := walkKey{from: ps, to: sid, kind: 2}
			if !savedStep[keyParentToQuai] {
				graph[ps] = append(graph[ps], types.Edge{
					FromSID:      ps,
					ToSID:        sid,
					Weight:       180,
					RID:          "",
					Wheelchair:   types.AccessibilityYes,
					BikesAllowed: types.AccessibilityYes,
				})
				savedStep[keyParentToQuai] = true
			}

			keyQuaiToParent := walkKey{from: sid, to: ps, kind: 2}
			if !savedStep[keyQuaiToParent] {
				graph[sid] = append(graph[sid], types.Edge{
					FromSID:      sid,
					ToSID:        ps,
					Weight:       180,
					RID:          "",
					Wheelchair:   types.AccessibilityYes,
					BikesAllowed: types.AccessibilityYes,
				})
				savedStep[keyQuaiToParent] = true
			}
		}
	}

	for fromSID, edges := range graph {
		graph[fromSID] = slices.Clip(edges)
	}

	return graph
}

func BuildReverseGraph(graph types.GraphType) types.GraphType {
	reverseGraph := make(types.GraphType, len(graph))
	for _, edges := range graph {
		for _, edge := range edges {
			revEdge := edge
			if len(edge.Schedules) > 1 {
				revEdge.Schedules = make([]types.DepartureTime, len(edge.Schedules))
				copy(revEdge.Schedules, edge.Schedules)
				slices.SortFunc(revEdge.Schedules, func(a, b types.DepartureTime) int {
					if a.Arr != b.Arr {
						return cmp.Compare(a.Arr, b.Arr)
					}
					return cmp.Compare(a.Dep, b.Dep)
				})
			}
			reverseGraph[edge.ToSID] = append(reverseGraph[edge.ToSID], revEdge)
		}
	}
	for sid, edges := range reverseGraph {
		reverseGraph[sid] = slices.Clip(edges)
	}
	return reverseGraph
}
