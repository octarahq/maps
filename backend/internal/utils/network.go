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

// EnrichRoutesWithVehicles assigns realistic IDFM vehicle illustrations to routes.
func EnrichRoutesWithVehicles(routes []types.Route) {
	for i := range routes {
		r := &routes[i]
		sname := strings.TrimSpace(r.SName)
		snameUpper := strings.ToUpper(sname)
		lnameUpper := strings.ToUpper(strings.TrimSpace(r.LName))

		switch {
		// 1. RER
		case (sname == "A" || sname == "B" || sname == "C" || sname == "D" || sname == "E") &&
			(r.RType == types.RouteTypeTrain || strings.Contains(lnameUpper, "RER")):
			switch sname {
			case "A":
				r.VehicleImage = "IDFM-rer-a.png"
			case "B":
				r.VehicleImage = "IDFM-rer-b.png"
				r.VehicleCarriage = "IDFM-rer-b-carriage.png"
			case "C":
				r.VehicleImage = "IDFM-rer-c.png"
			case "D":
				r.VehicleImage = "IDFM-rer-d.png"
			case "E":
				r.VehicleImage = "IDFM-rer-ng.png"
			}

		// 2. Transilien
		case (sname == "H" || sname == "J" || sname == "K" || sname == "L" || sname == "N" ||
			sname == "P" || sname == "R" || sname == "U" || sname == "V") && r.RType == types.RouteTypeTrain:
			switch sname {
			case "H", "J", "K", "L", "P":
				r.VehicleImage = "IDFM-train-francilien.png"
			case "N", "R":
				r.VehicleImage = "IDFM-train-regio2n.png"
			case "U":
				r.VehicleImage = "IDFM-rer-c.png"
			default:
				r.VehicleImage = "IDFM-train-ter.png"
			}

		// 3. Métro
		case r.RType == types.RouteTypeMetro || strings.Contains(lnameUpper, "MÉTRO") || strings.Contains(lnameUpper, "METRO"):
			switch sname {
			case "1":
				r.VehicleImage = "IDFM-metro-1.png"
			case "2":
				r.VehicleImage = "IDFM-metro-2.png"
			case "3", "3B", "3b":
				r.VehicleImage = "IDFM-metro-3.png"
			case "6":
				r.VehicleImage = "IDFM-metro-6.png"
				r.VehicleCarriage = "IDFM-metro-6-carriage.png"
			case "7", "7B", "7b":
				r.VehicleImage = "IDFM-metro-7.png"
			case "14":
				r.VehicleImage = "IDFM-metro-14.png"
				r.VehicleCarriage = "IDFM-metro-14-carriage.png"
			case "4", "11", "15", "16", "17", "18":
				r.VehicleImage = "IDFM-metro-mp14.png"
			case "5", "8", "9", "10", "12", "13":
				r.VehicleImage = "IDFM-metro-mf19.png"
				r.VehicleCarriage = "IDFM-metro-mf19-carriage.png"
			default:
				r.VehicleImage = "IDFM-metro-1.png"
			}

		// 4. Tramway
		case r.RType == types.RouteTypeTram:
			switch snameUpper {
			case "T1", "1":
				r.VehicleImage = "IDFM-tram-1.png"
			case "T2", "2":
				r.VehicleImage = "IDFM-tram-2.png"
			case "T3A", "T3B", "3A", "3B":
				r.VehicleImage = "IDFM-tram-3.png"
			case "T4", "4":
				r.VehicleImage = "IDFM-tram-4.png"
			case "T5", "5":
				r.VehicleImage = "IDFM-tram-5.png"
			case "T7", "7":
				r.VehicleImage = "IDFM-tram-7.png"
				r.VehicleCarriage = "IDFM-tram-7-carriage.png"
			case "T9", "T10", "9", "10":
				r.VehicleImage = "IDFM-tram-9.png"
			case "T11", "T12", "T13", "11", "12", "13":
				r.VehicleImage = "IDFM-tram-4.png" // Tram-train Citadis Dualis
			default:
				r.VehicleImage = "IDFM-tram-1.png"
			}

		// 5. Cable / Téléphérique
		case r.RType == types.RouteTypeAerialLift || r.RType == types.RouteTypeCableTram || r.RType == types.RouteTypeFunicular ||
			strings.Contains(lnameUpper, "CÂBLE") || strings.Contains(lnameUpper, "CABLE"):
			r.VehicleImage = "IDFM-cable.png"

		// 6. Bus
		case r.RType == types.RouteTypeBus:
			switch {
			case strings.Contains(snameUpper, "TZEN") || strings.Contains(lnameUpper, "TZEN") ||
				snameUpper == "TVM" || snameUpper == "393":
				r.VehicleImage = "IDFM-bus-tzen.png"
			case strings.HasPrefix(snameUpper, "N") && len(sname) <= 4:
				r.VehicleImage = "IDFM-bus-noctilien.png"
			case snameUpper == "PC" || snameUpper == "91" || snameUpper == "183" || snameUpper == "272" ||
				snameUpper == "26" || snameUpper == "38" || snameUpper == "43" || snameUpper == "62" ||
				snameUpper == "80" || snameUpper == "95" || snameUpper == "ROISSYBUS" || snameUpper == "ORLYBUS":
				r.VehicleImage = "IDFM-bus-articule.png"
			case strings.Contains(lnameUpper, "NAVETTE") || strings.Contains(lnameUpper, "ELECTRIQUE") ||
				strings.Contains(lnameUpper, "APTIS") || snameUpper == "TRAVERSIERE" || snameUpper == "TUB":
				r.VehicleImage = "IDFM-bus-electrique.png"
			default:
				r.VehicleImage = "IDFM-bus-default.png"
			}

		default:
			r.VehicleImage = "IDFM-bus-default.png"
		}
	}
}
