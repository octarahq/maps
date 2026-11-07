package utils

import (
	"cmp"
	"encoding/gob"
	"fmt"
	"os"
	"slices"
	"strings"

	"maps/backend/internal/types"
)

func LoadNetwork(path ...string) (*types.Network, error) {
	filePath := "data/processed/network.bin"
	if len(path) > 0 && path[0] != "" {
		filePath = path[0]
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var net types.Network
	dec := gob.NewDecoder(file)
	if err := dec.Decode(&net); err != nil {
		return nil, err
	}

	return &net, nil
}

func LoadExtra(path ...string) (*types.ExtraData, error) {
	filePath := "data/processed/extra.bin"
	if len(path) > 0 && path[0] != "" {
		filePath = path[0]
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var extra types.ExtraData
	dec := gob.NewDecoder(file)
	if err := dec.Decode(&extra); err != nil {
		return nil, err
	}

	return &extra, nil
}

func IndexStopsByID(stops []types.Stop) map[string]types.Stop {
	m := make(map[string]types.Stop, len(stops)*2)
	for _, s := range stops {
		m[s.SID] = s
		clean := IdRemoveIDFMPrefix(s.SID)
		if clean != s.SID {
			m[clean] = s
		}
	}
	return m
}

func IndexRoutesByID(routes []types.Route) map[string]types.Route {
	m := make(map[string]types.Route, len(routes)*2)
	for _, r := range routes {
		m[r.RID] = r
		clean := IdRemoveIDFMPrefix(r.RID)
		if clean != r.RID {
			m[clean] = r
		}
	}
	return m
}

func IndexTripsByID(trips []types.Trip) map[string]types.Trip {
	m := make(map[string]types.Trip, len(trips)*2)
	for _, t := range trips {
		m[t.TID] = t
		clean := IdRemoveIDFMPrefix(t.TID)
		if clean != t.TID {
			m[clean] = t
		}
	}
	return m
}

func IndexStopTimesByTripID(stopTimes []types.StopTime) map[string][]types.StopTime {
	m := make(map[string][]types.StopTime)
	for _, st := range stopTimes {
		m[st.TID] = append(m[st.TID], st)
	}
	return m
}

func IndexTransfersByFromStop(transfers []types.Transfer) map[string][]types.Transfer {
	m := make(map[string][]types.Transfer)
	for _, tr := range transfers {
		m[tr.FromSID] = append(m[tr.FromSID], tr)
	}
	return m
}

func IndexPositionInTrainsByTripKey(positionInTrain []types.PositionInTrain) map[string]types.PositionInTrain {
	m := make(map[string]types.PositionInTrain, len(positionInTrain))
	for _, pit := range positionInTrain {
		keyRaw := fmt.Sprintf("%s-%s;%s", pit.FromSID, pit.ToSID, pit.RID)
		m[keyRaw] = pit
		keyClean := fmt.Sprintf("%s-%s;%s", IdRemoveIDFMPrefix(pit.FromSID), IdRemoveIDFMPrefix(pit.ToSID), IdRemoveIDFMPrefix(pit.RID))
		if keyClean != keyRaw {
			m[keyClean] = pit
		}
	}
	return m
}

func EnrichParentsStations(net *types.Network) {
	if len(net.ParentsStations) == 0 {
		return
	}

	childrenByParent := make(map[string][]types.Stop, len(net.ParentsStations))
	stopsByCleanID := make(map[string]types.Stop, len(net.Stops))
	stopIDToParentID := make(map[string]string, len(net.Stops))

	for _, s := range net.Stops {
		cleanSID := IdRemoveIDFMPrefix(s.SID)
		stopsByCleanID[cleanSID] = s

		cleanPID := IdRemoveIDFMPrefix(s.ParentStation)
		if cleanPID != "" {
			childrenByParent[cleanPID] = append(childrenByParent[cleanPID], s)
			stopIDToParentID[s.SID] = cleanPID
			stopIDToParentID[cleanSID] = cleanPID
		} else {
			stopIDToParentID[s.SID] = cleanSID
			stopIDToParentID[cleanSID] = cleanSID
		}
	}

	tripToRoute := make(map[string]string, len(net.Trips))
	for _, t := range net.Trips {
		tripToRoute[t.TID] = t.RID
	}

	parentRouteIDs := make(map[string]map[string]struct{}, len(net.ParentsStations))
	for _, st := range net.StopTimes {
		pID, ok := stopIDToParentID[st.SId]
		if !ok {
			continue
		}
		rID, ok := tripToRoute[st.TID]
		if !ok || rID == "" {
			continue
		}
		routesSet := parentRouteIDs[pID]
		if routesSet == nil {
			routesSet = make(map[string]struct{})
			parentRouteIDs[pID] = routesSet
		}
		routesSet[rID] = struct{}{}
	}

	routesByID := IndexRoutesByID(net.Routes)

	for i := range net.ParentsStations {
		p := &net.ParentsStations[i]
		cleanPID := IdRemoveIDFMPrefix(p.PID)
		p.PID = cleanPID
		children := childrenByParent[cleanPID]

		if len(children) > 0 {
			p.ChildrenStops = make([]string, 0, len(children))
			for _, c := range children {
				p.ChildrenStops = append(p.ChildrenStops, IdRemoveIDFMPrefix(c.SID))
			}
		}

		switch len(children) {
		case 0:
			if st, ok := stopsByCleanID[cleanPID]; ok {
				p.Lat = st.Lat
				p.Lon = st.Long
			}
		case 1:
			p.Lat = children[0].Lat
			p.Lon = children[0].Long
		case 2:
			d := CoordsGetDistance(children[0].Long, children[0].Lat, children[1].Long, children[1].Lat)
			if d <= 0.030 {
				p.Lat = (children[0].Lat + children[1].Lat) / 2.0
				p.Lon = (children[0].Long + children[1].Long) / 2.0
			} else {
				p.Lat = children[0].Lat
				p.Lon = children[0].Long
			}
		default:
			var sumLat, sumLon float64
			for _, c := range children {
				sumLat += c.Lat
				sumLon += c.Long
			}
			p.Lat = sumLat / float64(len(children))
			p.Lon = sumLon / float64(len(children))
		}

		rIDs := parentRouteIDs[cleanPID]
		if len(rIDs) > 0 {
			type routeKey struct {
				name  string
				rtype types.RouteType
			}
			seenRoutes := make(map[routeKey]bool)
			p.Routes = make([]types.ParentsStationRoute, 0, len(rIDs))

			for rID := range rIDs {
				r, ok := routesByID[rID]
				if !ok {
					continue
				}
				name := r.SName
				if name == "" {
					name = r.LName
				}
				key := routeKey{name: name, rtype: r.RType}
				if seenRoutes[key] {
					continue
				}
				seenRoutes[key] = true

				p.Routes = append(p.Routes, types.ParentsStationRoute{
					RID:   IdRemoveIDFMPrefix(r.RID),
					Name:  name,
					Color: r.Color,
					RType: r.RType,
					Picto: r.Picto,
				})
			}

			slices.SortFunc(p.Routes, func(a, b types.ParentsStationRoute) int {
				if a.RType != b.RType {
					return cmp.Compare(a.RType, b.RType)
				}
				return strings.Compare(a.Name, b.Name)
			})
		}
	}
}

func EnrichRoutesWithPictos(routes []types.Route, lines []types.LineReferentiel) {
	if len(lines) == 0 || len(routes) == 0 {
		return
	}
	pictoMap := make(map[string]string, len(lines))
	for _, l := range lines {
		cleanID := IdRemoveIDFMPrefix(l.IDLine)
		if cleanID == "" {
			continue
		}
		pictoID := ExtractPictoID(l.Picto)
		if pictoID != "" {
			pictoMap[cleanID] = pictoID
		}
	}

	for i := range routes {
		cleanRID := IdRemoveIDFMPrefix(routes[i].RID)
		if picto, ok := pictoMap[cleanRID]; ok {
			routes[i].Picto = picto
		}
	}
}
