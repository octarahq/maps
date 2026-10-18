package utils

import (
	"fmt"
	"maps/api/internal/types"
)

func BuildGraph(net *types.Network) types.GraphType {
	graph := make(types.GraphType)
	trips := IndexTripsByID(net.Trips)
	routes := IndexRoutesByID(net.Routes)

	stopsTime := net.StopTimes
	savedStep := make(map[string]bool)

	for i := 1; i < len(stopsTime); i++ {
		prev := stopsTime[i-1]
		curr := stopsTime[i]

		if prev.TID == curr.TID {
			rid := trips[curr.TID].RID
			routeInfo := routes[rid]

			diff := int(curr.ATime) - int(prev.DTime)
			if diff <= 0 {
				diff = 30
			}

			key := fmt.Sprintf("%s-%s@%s", prev.SId, curr.SId, rid)
			if !savedStep[key] {
				graph[prev.SId] = append(graph[prev.SId], types.Edge{
					FromSID: prev.SId,
					ToSID:   curr.SId,
					Weight:  diff,
					RID:     routeInfo.RID,
				})
				savedStep[key] = true
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
				FromSID: tr.FromSID,
				ToSID:   tr.ToSID,
				Weight:  w,
				RID:     "",
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
							FromSID: from,
							ToSID:   to,
							Weight:  180,
							RID:     "",
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
					FromSID: s.ParentStation,
					ToSID:   s.SID,
					Weight:  180,
					RID:     "",
				})
				savedStep[keyParentToQuai] = true
			}

			keyQuaiToParent := fmt.Sprintf("%s-%s@zdc", s.SID, s.ParentStation)
			if !savedStep[keyQuaiToParent] {
				graph[s.SID] = append(graph[s.SID], types.Edge{
					FromSID: s.SID,
					ToSID:   s.ParentStation,
					Weight:  180,
					RID:     "",
				})
				savedStep[keyQuaiToParent] = true
			}
		}
	}

	return graph
}
