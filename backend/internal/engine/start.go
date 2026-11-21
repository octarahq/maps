package engine

import (
	"log"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"time"

	"maps/backend/internal/types"
	"maps/backend/internal/utils"
)

type ShapeEntry struct {
	ID             string
	Points         [][2]float64
	MinLat, MaxLat float64
	MinLon, MaxLon float64
}

type Engine struct {
	Net                 types.Network
	Graph               types.GraphType
	ReverseGraph        types.GraphType
	Extras              types.ExtraData
	CalendarIndex       *utils.CalendarIndex
	Disruptions         []types.Disruption
	disruptionsMu       *sync.RWMutex
	delays              *DelaysData
	delaysMu            *sync.RWMutex
	stopMonCache        *stopMonitoringCache
	tripIdxByUUID       map[string][]uint32
	tripIdxByRouteSName map[string][]uint32
	parentsByChild      map[string]string
	childrenByParent    map[string][]string
	rawSIDByClean       map[string]string
	shapesIndex         map[string][][2]float64
	shapesByType        map[types.RouteType][]ShapeEntry
	routesByID          map[string]types.Route
	StopTimesCount      int
}

func StartEngine() Engine {
	var e Engine
	e.disruptionsMu = &sync.RWMutex{}
	e.delaysMu = &sync.RWMutex{}
	e.stopMonCache = newStopMonitoringCache()
	e.parentsByChild = make(map[string]string)
	e.childrenByParent = make(map[string][]string)
	e.rawSIDByClean = make(map[string]string)
	net, err := utils.LoadNetwork()
	if err != nil {
		log.Fatal(err)
	}
	e.Net = *net
	net = nil
	e.StopTimesCount = len(e.Net.StopTimes)

	utils.EnrichRoutesWithVehicles(e.Net.Routes)
	utils.EnrichTripsHeadsignWithTerminus(&e.Net)

	internPool := make(map[string]string, 65536)
	intern := func(s string) string {
		if s == "" {
			return ""
		}
		if v, ok := internPool[s]; ok {
			return v
		}
		internPool[s] = s
		return s
	}

	for i := range e.Net.Routes {
		r := &e.Net.Routes[i]
		r.RID = intern(r.RID)
		r.AgencyID = intern(r.AgencyID)
		r.Color = intern(r.Color)
		r.TColor = intern(r.TColor)
		r.Picto = intern(r.Picto)
		r.VehicleImage = intern(r.VehicleImage)
		r.VehicleCarriage = intern(r.VehicleCarriage)
	}
	e.routesByID = utils.IndexRoutesByID(e.Net.Routes)

	for i := range e.Net.Stops {
		s := &e.Net.Stops[i]
		s.SID = intern(s.SID)
		s.ZoneId = intern(s.ZoneId)
		s.ParentStation = intern(s.ParentStation)
		clean := intern(utils.IdRemoveIDFMPrefix(s.SID))
		e.rawSIDByClean[clean] = s.SID
		e.rawSIDByClean[s.SID] = s.SID
	}
	for _, ps := range e.Net.ParentsStations {
		clean := intern(utils.IdRemoveIDFMPrefix(ps.PID))
		if _, ok := e.rawSIDByClean[clean]; !ok {
			e.rawSIDByClean[clean] = ps.PID
		}
		if _, ok := e.rawSIDByClean[ps.PID]; !ok {
			e.rawSIDByClean[ps.PID] = ps.PID
		}
	}

	if len(e.Net.ParentsStations) > 0 && e.Net.ParentsStations[0].Lat == 0 && e.Net.ParentsStations[0].Lon == 0 {
		utils.EnrichParentsStations(&e.Net)
	}

	for i := range e.Net.ParentsStations {
		p := &e.Net.ParentsStations[i]
		p.PID = intern(utils.IdRemoveIDFMPrefix(p.PID))
		for j := range p.ChildrenStops {
			p.ChildrenStops[j] = intern(utils.IdRemoveIDFMPrefix(p.ChildrenStops[j]))
		}
		p.ChildrenStops = slices.Clip(p.ChildrenStops)
		for j := range p.Routes {
			r := &p.Routes[j]
			r.RID = intern(utils.IdRemoveIDFMPrefix(r.RID))
			r.Name = intern(r.Name)
			r.Color = intern(r.Color)
			r.Picto = intern(r.Picto)
		}
		p.Routes = slices.Clip(p.Routes)
	}

	for _, p := range e.Net.ParentsStations {
		cleanPID := p.PID
		e.childrenByParent[cleanPID] = p.ChildrenStops
		for _, c := range p.ChildrenStops {
			e.parentsByChild[c] = cleanPID
		}
	}
	for _, s := range e.Net.Stops {
		cleanSID := intern(utils.IdRemoveIDFMPrefix(s.SID))
		cleanPID := intern(utils.IdRemoveIDFMPrefix(s.ParentStation))
		if cleanPID != "" {
			e.parentsByChild[cleanSID] = cleanPID
			e.childrenByParent[cleanPID] = append(e.childrenByParent[cleanPID], cleanSID)
		}
	}
	for k, v := range e.childrenByParent {
		e.childrenByParent[k] = slices.Clip(v)
	}

	for i := range e.Net.PositionsInTrain {
		p := &e.Net.PositionsInTrain[i]
		p.FromSID = intern(p.FromSID)
		p.ToSID = intern(p.ToSID)
		p.RID = intern(p.RID)
	}

	extra, err := utils.LoadExtra()
	if err == nil && extra != nil {
		e.Extras = *extra
		extra = nil
	}

	for i := range e.Extras.BikeParkings {
		e.Extras.BikeParkings[i].Name = intern(e.Extras.BikeParkings[i].Name)
	}

	e.shapesIndex = make(map[string][][2]float64, 15000)
	for _, s := range e.Extras.Shapes {
		sid := intern(s.ShapeID)
		e.shapesIndex[sid] = append(e.shapesIndex[sid], [2]float64{s.Lat, s.Lon})
	}
	for id, pts := range e.shapesIndex {
		pts = slices.Clip(pts)
		e.shapesIndex[id] = pts
		clean := intern(utils.IdRemoveIDFMPrefix(id))
		if clean != id {
			e.shapesIndex[clean] = pts
		}
	}

	e.Extras.Shapes = nil

	e.shapesByType = make(map[types.RouteType][]ShapeEntry)
	for id, pts := range e.shapesIndex {
		if len(pts) == 0 || !strings.HasPrefix(id, "IDFM:") {
			continue
		}
		parts := strings.Split(id, "_")
		if len(parts) >= 2 {
			var rType types.RouteType
			switch parts[1] {
			case "0":
				rType = types.RouteTypeTram
			case "1":
				rType = types.RouteTypeMetro
			case "2":
				rType = types.RouteTypeTrain
			case "3":
				rType = types.RouteTypeBus
			case "4":
				rType = types.RouteTypeFerry
			case "6":
				rType = types.RouteTypeAerialLift
			case "7":
				rType = types.RouteTypeFunicular
			default:
				continue
			}
			minLat, maxLat := pts[0][0], pts[0][0]
			minLon, maxLon := pts[0][1], pts[0][1]
			for _, p := range pts {
				if p[0] < minLat {
					minLat = p[0]
				}
				if p[0] > maxLat {
					maxLat = p[0]
				}
				if p[1] < minLon {
					minLon = p[1]
				}
				if p[1] > maxLon {
					maxLon = p[1]
				}
			}
			e.shapesByType[rType] = append(e.shapesByType[rType], ShapeEntry{
				ID:     id,
				Points: pts,
				MinLat: minLat,
				MaxLat: maxLat,
				MinLon: minLon,
				MaxLon: maxLon,
			})
		}
	}

	if len(e.Net.Trips) > 0 && e.Net.Trips[0].ShapeID == "" {
		if mapping, err := utils.LoadTripShapeMapping("data/processed/trips.txt"); err == nil {
			for i := range e.Net.Trips {
				if shpID, ok := mapping[e.Net.Trips[i].TID]; ok {
					e.Net.Trips[i].ShapeID = shpID
				}
			}
		}
	}

	if len(e.Net.Trips) > 0 && e.Net.Trips[0].ServiceID == "" {
		if mapping, err := utils.LoadTripServiceMapping("data/raw/trips.txt"); err == nil {
			for i := range e.Net.Trips {
				if sID, ok := mapping[e.Net.Trips[i].TID]; ok {
					e.Net.Trips[i].ServiceID = sID
				}
			}
		}
	}

	e.CalendarIndex = utils.NewCalendarIndex(e.Extras.Calendars, e.Extras.CalendarDates)

	e.Extras.Agencies = nil
	e.Extras.Calendars = nil
	e.Extras.CalendarDates = nil
	e.Extras.Pathways = nil
	e.Extras.Attributions = nil
	e.Extras.BookingRules = nil
	e.Extras.ObjectCodesExtension = nil
	e.Extras.TicketingDeepLinks = nil

	for i := range e.Net.Trips {
		t := &e.Net.Trips[i]
		t.RID = intern(t.RID)
		t.ServiceID = intern(t.ServiceID)
		t.Headsign = intern(t.Headsign)
		t.SName = intern(t.SName)
		t.ShapeID = intern(t.ShapeID)
		if t.ServiceID != "" {
			e.CalendarIndex.GetOrRegisterService(t.ServiceID)
		}
	}
	internPool = nil

	e.Graph = utils.BuildGraph(&e.Net, e.CalendarIndex)

	e.Net.StopTimes = nil
	e.Net.Transfers = nil
	runtime.GC()

	e.ReverseGraph = utils.BuildReverseGraph(e.Graph)
	e.buildTripLookupIndices()
	runtime.GC()
	debug.FreeOSMemory()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		e.FetchDisruptions()
	}()
	go func() {
		defer wg.Done()
		e.FetchDelays()
	}()
	wg.Wait()

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			e.FetchDisruptions()
		}
	}()

	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			e.FetchDelays()
		}
	}()

	return e
}

func (e *Engine) GetActiveServices(target time.Time) []bool {
	if e.CalendarIndex == nil {
		return nil
	}
	return e.CalendarIndex.GetActiveServices(target)
}

func (e *Engine) ResolveStopSID(id string) string {
	if raw, ok := e.rawSIDByClean[id]; ok {
		return raw
	}
	clean := utils.IdRemoveIDFMPrefix(id)
	if raw, ok := e.rawSIDByClean[clean]; ok {
		return raw
	}
	if len(e.Graph[id]) > 0 {
		return id
	}
	if len(e.Graph["IDFM:"+id]) > 0 {
		return "IDFM:" + id
	}
	if len(e.Graph["IDFM:monomodalStopPlace:"+clean]) > 0 {
		return "IDFM:monomodalStopPlace:" + clean
	}
	if len(e.Graph["IDFM:"+clean]) > 0 {
		return "IDFM:" + clean
	}
	if len(e.Graph[clean]) > 0 {
		return clean
	}
	return id
}

func (e *Engine) GetEdgesForStop(id string) []types.Edge {
	raw := e.ResolveStopSID(id)
	if edges, ok := e.Graph[raw]; ok && len(edges) > 0 {
		return edges
	}
	if edges, ok := e.Graph[id]; ok && len(edges) > 0 {
		return edges
	}
	clean := utils.IdRemoveIDFMPrefix(id)
	if edges, ok := e.Graph["IDFM:"+clean]; ok && len(edges) > 0 {
		return edges
	}
	if edges, ok := e.Graph[clean]; ok && len(edges) > 0 {
		return edges
	}
	return nil
}

func (e *Engine) GetReverseEdgesForStop(id string) []types.Edge {
	raw := e.ResolveStopSID(id)
	if edges, ok := e.ReverseGraph[raw]; ok && len(edges) > 0 {
		return edges
	}
	if edges, ok := e.ReverseGraph[id]; ok && len(edges) > 0 {
		return edges
	}
	clean := utils.IdRemoveIDFMPrefix(id)
	if edges, ok := e.ReverseGraph["IDFM:"+clean]; ok && len(edges) > 0 {
		return edges
	}
	if edges, ok := e.ReverseGraph[clean]; ok && len(edges) > 0 {
		return edges
	}
	return nil
}

func (e *Engine) GetShapesIndex() map[string][][2]float64 {
	return e.shapesIndex
}
