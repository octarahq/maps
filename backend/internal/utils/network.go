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
	if err := gob.NewDecoder(file).Decode(&net); err != nil {
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
	if err := gob.NewDecoder(file).Decode(&extra); err != nil {
		return nil, err
	}
	return &extra, nil
}

func IndexStopsByID(stops []types.Stop) map[string]types.Stop {
	m := make(map[string]types.Stop, len(stops)*2)
	for _, s := range stops {
		m[s.SID] = s
		if clean := IdRemoveIDFMPrefix(s.SID); clean != s.SID {
			m[clean] = s
		}
	}
	return m
}

func IndexRoutesByID(routes []types.Route) map[string]types.Route {
	m := make(map[string]types.Route, len(routes)*2)
	for _, r := range routes {
		m[r.RID] = r
		if clean := IdRemoveIDFMPrefix(r.RID); clean != r.RID {
			m[clean] = r
		}
	}
	return m
}

func IndexTripsByID(trips []types.Trip) map[string]types.Trip {
	m := make(map[string]types.Trip, len(trips)*2)
	for _, t := range trips {
		m[t.TID] = t
		if clean := IdRemoveIDFMPrefix(t.TID); clean != t.TID {
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
		targetPID := cleanSID
		if cleanPID != "" {
			childrenByParent[cleanPID] = append(childrenByParent[cleanPID], s)
			targetPID = cleanPID
		}
		stopIDToParentID[s.SID] = targetPID
		stopIDToParentID[cleanSID] = targetPID
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
		rID := tripToRoute[st.TID]
		if rID == "" {
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
			p.ChildrenStops = make([]string, len(children))
			for j, c := range children {
				p.ChildrenStops[j] = IdRemoveIDFMPrefix(c.SID)
			}
		}

		switch len(children) {
		case 0:
			if st, ok := stopsByCleanID[cleanPID]; ok {
				p.Lat, p.Lon = st.Lat, st.Long
			}
		case 1:
			p.Lat, p.Lon = children[0].Lat, children[0].Long
		case 2:
			if CoordsGetDistance(children[0].Long, children[0].Lat, children[1].Long, children[1].Lat) <= 0.030 {
				p.Lat = (children[0].Lat + children[1].Lat) / 2.0
				p.Lon = (children[0].Long + children[1].Long) / 2.0
			} else {
				p.Lat, p.Lon = children[0].Lat, children[0].Long
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
		if pictoID := ExtractPictoID(l.Picto); pictoID != "" {
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

type vehicleInfo struct {
	image    string
	carriage string
}

var (
	rerVehicles = map[string]vehicleInfo{
		"A": {image: "IDFM-rer-a.png"},
		"B": {image: "IDFM-rer-b.png", carriage: "IDFM-rer-b-carriage.png"},
		"C": {image: "IDFM-rer-c.png"},
		"D": {image: "IDFM-rer-d.png"},
		"E": {image: "IDFM-rer-ng.png"},
	}

	trainVehicles = map[string]vehicleInfo{
		"H": {image: "IDFM-train-francilien.png"},
		"J": {image: "IDFM-train-francilien.png"},
		"K": {image: "IDFM-train-francilien.png"},
		"L": {image: "IDFM-train-francilien.png"},
		"P": {image: "IDFM-train-francilien.png"},
		"N": {image: "IDFM-train-regio2n.png"},
		"R": {image: "IDFM-train-regio2n.png"},
		"U": {image: "IDFM-rer-c.png"},
	}

	metroVehicles = map[string]vehicleInfo{
		"1":  {image: "IDFM-metro-1.png"},
		"2":  {image: "IDFM-metro-2.png"},
		"3":  {image: "IDFM-metro-3.png"},
		"3B": {image: "IDFM-metro-3.png"},
		"6":  {image: "IDFM-metro-6.png", carriage: "IDFM-metro-6-carriage.png"},
		"7":  {image: "IDFM-metro-7.png"},
		"7B": {image: "IDFM-metro-7.png"},
		"14": {image: "IDFM-metro-14.png", carriage: "IDFM-metro-14-carriage.png"},
		"4":  {image: "IDFM-metro-mp14.png"},
		"11": {image: "IDFM-metro-mp14.png"},
		"15": {image: "IDFM-metro-mp14.png"},
		"16": {image: "IDFM-metro-mp14.png"},
		"17": {image: "IDFM-metro-mp14.png"},
		"18": {image: "IDFM-metro-mp14.png"},
		"5":  {image: "IDFM-metro-mf19.png", carriage: "IDFM-metro-mf19-carriage.png"},
		"8":  {image: "IDFM-metro-mf19.png", carriage: "IDFM-metro-mf19-carriage.png"},
		"9":  {image: "IDFM-metro-mf19.png", carriage: "IDFM-metro-mf19-carriage.png"},
		"10": {image: "IDFM-metro-mf19.png", carriage: "IDFM-metro-mf19-carriage.png"},
		"12": {image: "IDFM-metro-mf19.png", carriage: "IDFM-metro-mf19-carriage.png"},
		"13": {image: "IDFM-metro-mf19.png", carriage: "IDFM-metro-mf19-carriage.png"},
	}

	tramVehicles = map[string]vehicleInfo{
		"T1": {image: "IDFM-tram-1.png"}, "1": {image: "IDFM-tram-1.png"},
		"T2": {image: "IDFM-tram-2.png"}, "2": {image: "IDFM-tram-2.png"},
		"T3A": {image: "IDFM-tram-3.png"}, "3A": {image: "IDFM-tram-3.png"},
		"T3B": {image: "IDFM-tram-3.png"}, "3B": {image: "IDFM-tram-3.png"},
		"T4": {image: "IDFM-tram-4.png"}, "4": {image: "IDFM-tram-4.png"},
		"T5": {image: "IDFM-tram-5.png"}, "5": {image: "IDFM-tram-5.png"},
		"T7": {image: "IDFM-tram-7.png", carriage: "IDFM-tram-7-carriage.png"}, "7": {image: "IDFM-tram-7.png", carriage: "IDFM-tram-7-carriage.png"},
		"T9": {image: "IDFM-tram-9.png"}, "9": {image: "IDFM-tram-9.png"},
		"T10": {image: "IDFM-tram-9.png"}, "10": {image: "IDFM-tram-9.png"},
		"T11": {image: "IDFM-tram-4.png"}, "11": {image: "IDFM-tram-4.png"},
		"T12": {image: "IDFM-tram-4.png"}, "12": {image: "IDFM-tram-4.png"},
		"T13": {image: "IDFM-tram-4.png"}, "13": {image: "IDFM-tram-4.png"},
	}

	busArticulatedLines = map[string]bool{
		"PC": true, "91": true, "183": true, "272": true, "26": true, "38": true,
		"43": true, "62": true, "80": true, "95": true, "ROISSYBUS": true, "ORLYBUS": true,
	}
)

func resolveVehicleForRoute(r *types.Route) (image, carriage string) {
	sname := strings.TrimSpace(r.SName)
	snameUpper := strings.ToUpper(sname)
	lnameUpper := strings.ToUpper(strings.TrimSpace(r.LName))

	if (sname == "A" || sname == "B" || sname == "C" || sname == "D" || sname == "E") &&
		(r.RType == types.RouteTypeTrain || strings.Contains(lnameUpper, "RER")) {
		if v, ok := rerVehicles[sname]; ok {
			return v.image, v.carriage
		}
	}

	if (sname == "H" || sname == "J" || sname == "K" || sname == "L" || sname == "N" ||
		sname == "P" || sname == "R" || sname == "U" || sname == "V") && r.RType == types.RouteTypeTrain {
		if v, ok := trainVehicles[sname]; ok {
			return v.image, v.carriage
		}
		return "IDFM-train-ter.png", ""
	}

	if r.RType == types.RouteTypeMetro || strings.Contains(lnameUpper, "MÉTRO") || strings.Contains(lnameUpper, "METRO") {
		if v, ok := metroVehicles[snameUpper]; ok {
			return v.image, v.carriage
		}
		return "IDFM-metro-1.png", ""
	}

	if r.RType == types.RouteTypeTram {
		if v, ok := tramVehicles[snameUpper]; ok {
			return v.image, v.carriage
		}
		return "IDFM-tram-1.png", ""
	}

	if r.RType == types.RouteTypeAerialLift || r.RType == types.RouteTypeCableTram || r.RType == types.RouteTypeFunicular ||
		strings.Contains(lnameUpper, "CÂBLE") || strings.Contains(lnameUpper, "CABLE") {
		return "IDFM-cable.png", ""
	}

	if r.RType == types.RouteTypeBus {
		switch {
		case strings.Contains(snameUpper, "TZEN") || strings.Contains(lnameUpper, "TZEN") ||
			snameUpper == "TVM" || snameUpper == "393":
			return "IDFM-bus-tzen.png", ""
		case strings.HasPrefix(snameUpper, "N") && len(sname) <= 4:
			return "IDFM-bus-noctilien.png", ""
		case busArticulatedLines[snameUpper]:
			return "IDFM-bus-articule.png", ""
		case strings.Contains(lnameUpper, "NAVETTE") || strings.Contains(lnameUpper, "ELECTRIQUE") ||
			strings.Contains(lnameUpper, "APTIS") || snameUpper == "TRAVERSIERE" || snameUpper == "TUB":
			return "IDFM-bus-electrique.png", ""
		default:
			return "IDFM-bus-default.png", ""
		}
	}

	return "IDFM-bus-default.png", ""
}

func EnrichRoutesWithVehicles(routes []types.Route) {
	for i := range routes {
		routes[i].VehicleImage, routes[i].VehicleCarriage = resolveVehicleForRoute(&routes[i])
	}
}

func EnrichTripsHeadsignWithTerminus(net *types.Network) {
	if len(net.Trips) == 0 || len(net.StopTimes) == 0 {
		return
	}

	type tripLast struct {
		maxSeq uint16
		sid    string
	}
	lastStops := make(map[string]tripLast, len(net.Trips))
	for _, st := range net.StopTimes {
		cur, ok := lastStops[st.TID]
		if !ok || st.Seq >= cur.maxSeq {
			lastStops[st.TID] = tripLast{maxSeq: st.Seq, sid: st.SId}
		}
	}

	stopsByID := IndexStopsByID(net.Stops)

	for i := range net.Trips {
		t := &net.Trips[i]
		ls, ok := lastStops[t.TID]
		if !ok {
			cleanTID := IdRemoveIDFMPrefix(t.TID)
			ls, ok = lastStops[cleanTID]
		}
		if ok {
			if stop, sOk := stopsByID[ls.sid]; sOk && stop.Name != "" {
				t.Headsign = stop.Name
			}
		}
	}
}
