package engine

import (
	"log"
	"maps/api/internal/types"
	"maps/api/internal/utils"
)

type Engine struct {
	Net          types.Network
	Graph        types.GraphType
	StationStops []types.Stop
	BikeParkings []types.BikeParking
}

func StartEngine() Engine {
	var e Engine
	net, err := utils.LoadNetwork()
	if err != nil {
		log.Fatal(err)
	}
	e.Net = *net

	e.Graph = utils.BuildGraph(net)

	stationPIDs := make(map[string]bool)
	for _, ps := range net.ParentsStations {
		if ps.PType == types.ParentTypeRail || ps.PType == types.ParentTypeMetro || ps.PType == types.ParentTypeStreetTram {
			stationPIDs[ps.PID] = true
			stationPIDs["IDFM:"+ps.PID] = true
		}
	}

	for _, s := range net.Stops {
		if len(e.Graph[s.SID]) == 0 {
			continue
		}
		pid := s.ParentStation
		if pid == "" {
			pid = s.SID
		}
		if stationPIDs[pid] {
			e.StationStops = append(e.StationStops, s)
		}
	}

	extra, err := utils.LoadExtra()
	if err == nil && extra != nil {
		e.BikeParkings = extra.BikeParkings
	}

	return e
}
