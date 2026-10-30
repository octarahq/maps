package engine

import (
	"math"
	"sort"
	"strings"

	"maps/backend/internal/types"
	"maps/backend/internal/utils"
)

func normalize(s string) string {
	s = strings.ToLower(s)
	replacer := strings.NewReplacer(
		"é", "e", "è", "e", "ê", "e", "ë", "e",
		"à", "a", "â", "a", "ä", "a",
		"î", "i", "ï", "i",
		"ô", "o", "ö", "o",
		"ù", "u", "û", "u", "ü", "u",
		"ç", "c",
		"-", " ",
		"'", " ",
	)
	return replacer.Replace(s)
}

func (e *Engine) SearchParentsStopsByName(query string) []types.ParentsStation {
	normQ := strings.TrimSpace(normalize(query))
	if normQ == "" {
		return nil
	}

	type candidate struct {
		stop  types.ParentsStation
		score int
	}

	var candidates []candidate
	seenNames := make(map[string]bool)

	for _, stop := range e.Net.ParentsStations {
		normName := normalize(stop.Name)

		if normName == normQ {
			candidates = append(candidates, candidate{stop: stop, score: 0})
			continue
		}

		if strings.HasPrefix(normName, normQ) {
			candidates = append(candidates, candidate{stop: stop, score: 10})
			continue
		}

		if strings.Contains(normName, normQ) {
			candidates = append(candidates, candidate{stop: stop, score: 20})
			continue
		}

		words := strings.Fields(normName)
		bestWordDist := 999
		for _, w := range words {
			if strings.HasPrefix(w, normQ) {
				bestWordDist = 15
				break
			}
			diff := len(w) - len(normQ)
			if diff < 0 {
				diff = -diff
			}
			if diff <= 2 {
				d := utils.LevenshteinDistance(w, normQ)
				if d < bestWordDist {
					bestWordDist = d
				}
			}
		}

		if bestWordDist <= 2 {
			candidates = append(candidates, candidate{stop: stop, score: 100 + bestWordDist})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score < candidates[j].score
		}
		return len(candidates[i].stop.Name) < len(candidates[j].stop.Name)
	})

	var result []types.ParentsStation
	for _, c := range candidates {
		if !seenNames[c.stop.Name] {
			seenNames[c.stop.Name] = true
			result = append(result, c.stop)
		}
	}

	return result
}

func (e *Engine) SearchStopsByName(query string) []types.Stop {
	normQ := strings.TrimSpace(normalize(query))
	if normQ == "" {
		return nil
	}

	type candidate struct {
		stop  types.Stop
		score int
	}

	var candidates []candidate
	seenNames := make(map[string]bool)

	for _, stop := range e.Net.Stops {
		normName := normalize(stop.Name)

		if normName == normQ {
			candidates = append(candidates, candidate{stop: stop, score: 0})
			continue
		}

		if strings.HasPrefix(normName, normQ) {
			candidates = append(candidates, candidate{stop: stop, score: 10})
			continue
		}

		if strings.Contains(normName, normQ) {
			candidates = append(candidates, candidate{stop: stop, score: 20})
			continue
		}

		words := strings.Fields(normName)
		bestWordDist := 999
		for _, w := range words {
			if strings.HasPrefix(w, normQ) {
				bestWordDist = 15
				break
			}
			diff := len(w) - len(normQ)
			if diff < 0 {
				diff = -diff
			}
			if diff <= 2 {
				d := utils.LevenshteinDistance(w, normQ)
				if d < bestWordDist {
					bestWordDist = d
				}
			}
		}

		if bestWordDist <= 2 {
			candidates = append(candidates, candidate{stop: stop, score: 100 + bestWordDist})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score < candidates[j].score
		}
		return len(candidates[i].stop.Name) < len(candidates[j].stop.Name)
	})

	var result []types.Stop
	for _, c := range candidates {
		if !seenNames[c.stop.Name] {
			seenNames[c.stop.Name] = true
			result = append(result, c.stop)
		}
	}

	return result
}

func (e *Engine) SearchStopById(query string) (types.Stop, bool) {
	for _, stop := range e.Net.Stops {
		if utils.IdRemoveIDFMPrefix(stop.SID) == utils.IdRemoveIDFMPrefix(query) {
			return stop, true
		}
	}

	return types.Stop{}, false
}

func (e *Engine) SearchStopsByNeerCord(lon, lat float64, maxK int) []types.Stop {
	type nearbyStop struct {
		stop     types.Stop
		distance float64
	}

	stops := make([]nearbyStop, 0, len(e.Net.Stops))
	for _, stop := range e.Net.Stops {
		if stop.ParentStation != "" {
			continue
		}
		distance := utils.CoordsGetDistance(stop.Long, stop.Lat, lon, lat)

		stops = append(stops, nearbyStop{stop: stop, distance: distance})
	}

	sort.Slice(stops, func(i, j int) bool {
		return stops[i].distance < stops[j].distance
	})

	if maxK > 0 && maxK < len(stops) {
		stops = stops[:maxK]
	}

	result := make([]types.Stop, 0, len(stops))
	for _, nearby := range stops {
		result = append(result, nearby.stop)
	}

	return result
}

func (e *Engine) SearchBoardingStopsByCoord(lon, lat float64, maxDistanceKm float64, maxK int) []types.Stop {
	type nearbyStop struct {
		stop     types.Stop
		distance float64
	}

	latDelta := maxDistanceKm / 110.0
	lonDelta := maxDistanceKm / 70.0

	var stops []nearbyStop
	for _, stop := range e.Net.Stops {
		if math.Abs(stop.Lat-lat) > latDelta || math.Abs(stop.Long-lon) > lonDelta {
			continue
		}
		edges := e.Graph[stop.SID]
		if len(edges) == 0 {
			continue
		}
		hasTransit := false
		for _, ed := range edges {
			if ed.RID != "" {
				hasTransit = true
				break
			}
		}
		if !hasTransit {
			continue
		}

		dist := utils.CoordsGetDistance(stop.Long, stop.Lat, lon, lat)
		if dist <= maxDistanceKm {
			stops = append(stops, nearbyStop{stop: stop, distance: dist})
		}
	}

	sort.Slice(stops, func(i, j int) bool {
		return stops[i].distance < stops[j].distance
	})

	if maxK > 0 && maxK < len(stops) {
		stops = stops[:maxK]
	}

	result := make([]types.Stop, 0, len(stops))
	for _, nearby := range stops {
		result = append(result, nearby.stop)
	}

	return result
}

func (e *Engine) SearchArrivalStopsByCoord(lon, lat float64, maxDistanceKm float64, maxK int) []types.Stop {
	type nearbyStop struct {
		stop     types.Stop
		distance float64
	}

	latDelta := maxDistanceKm / 110.0
	lonDelta := maxDistanceKm / 70.0

	var stops []nearbyStop
	for _, stop := range e.Net.Stops {
		if math.Abs(stop.Lat-lat) > latDelta || math.Abs(stop.Long-lon) > lonDelta {
			continue
		}
		if len(e.Graph[stop.SID]) == 0 {
			continue
		}
		dist := utils.CoordsGetDistance(stop.Long, stop.Lat, lon, lat)
		if dist <= maxDistanceKm {
			stops = append(stops, nearbyStop{stop: stop, distance: dist})
		}
	}

	sort.Slice(stops, func(i, j int) bool {
		return stops[i].distance < stops[j].distance
	})

	if maxK > 0 && maxK < len(stops) {
		stops = stops[:maxK]
	}

	result := make([]types.Stop, 0, len(stops))
	for _, nearby := range stops {
		result = append(result, nearby.stop)
	}

	return result
}

func (e *Engine) StationStops() []types.Stop {
	stationPIDs := make(map[string]bool, len(e.Net.ParentsStations)*2)
	for _, ps := range e.Net.ParentsStations {
		if ps.PType == types.ParentTypeRail || ps.PType == types.ParentTypeMetro || ps.PType == types.ParentTypeStreetTram {
			stationPIDs[ps.PID] = true
			stationPIDs["IDFM:"+ps.PID] = true
		}
	}

	var res []types.Stop
	for _, s := range e.Net.Stops {
		if len(e.Graph[s.SID]) == 0 {
			continue
		}
		pid := s.ParentStation
		if pid == "" {
			pid = s.SID
		}
		if stationPIDs[pid] {
			res = append(res, s)
		}
	}
	return res
}

func (e *Engine) SearchNearestStationByCoord(lon, lat float64, maxDistanceKm float64) (types.Stop, bool) {
	var bestStop types.Stop
	bestDist := math.MaxFloat64
	found := false

	for _, s := range e.StationStops() {
		dist := utils.CoordsGetDistance(s.Long, s.Lat, lon, lat)
		if dist <= maxDistanceKm && dist < bestDist {
			bestDist = dist
			bestStop = s
			found = true
		}
	}

	return bestStop, found
}

func (e *Engine) FindNearestBikeParking(lon, lat float64, maxDistanceKm float64) (*types.BikeParking, float64, bool) {
	if len(e.Extras.BikeParkings) == 0 {
		return nil, 0, false
	}
	var best *types.BikeParking
	bestDist := math.MaxFloat64

	latDelta := maxDistanceKm / 110.0
	lonDelta := maxDistanceKm / 70.0

	for i := range e.Extras.BikeParkings {
		bp := &e.Extras.BikeParkings[i]
		if math.Abs(bp.Lat-lat) > latDelta || math.Abs(bp.Lon-lon) > lonDelta {
			continue
		}
		dist := utils.CoordsGetDistance(lon, lat, bp.Lon, bp.Lat)
		if dist <= maxDistanceKm && dist < bestDist {
			bestDist = dist
			best = bp
		}
	}

	if best != nil {
		return best, bestDist, true
	}
	return nil, 0, false
}

func (e *Engine) SearchNearestStationWithBikeParkingByCoord(lon, lat float64, maxDistanceKm float64, maxParkingDistKm float64) (types.Stop, *types.BikeParking, bool) {
	var bestStop types.Stop
	var bestParking *types.BikeParking
	bestDist := math.MaxFloat64

	for _, s := range e.StationStops() {
		dist := utils.CoordsGetDistance(s.Long, s.Lat, lon, lat)
		if dist <= maxDistanceKm && dist < bestDist {
			bp, _, ok := e.FindNearestBikeParking(s.Long, s.Lat, maxParkingDistKm)
			if !ok {
				continue
			}
			bestDist = dist
			bestStop = s
			bestParking = bp
		}
	}

	if bestParking != nil {
		return bestStop, bestParking, true
	}
	return bestStop, nil, false
}
