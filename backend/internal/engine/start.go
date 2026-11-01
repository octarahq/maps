package engine

import (
	"log"
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
	Net              types.Network
	Graph            types.GraphType
	ReverseGraph     types.GraphType
	Extras           types.ExtraData
	CalendarIndex    *utils.CalendarIndex
	Disruptions      []types.Disruption
	disruptionsMu    *sync.RWMutex
	parentsByChild   map[string]string
	childrenByParent map[string][]string
	shapesIndex      map[string][][2]float64
	shapesByType     map[types.RouteType][]ShapeEntry
}


func StartEngine() Engine {
	var e Engine
	e.disruptionsMu = &sync.RWMutex{}
	e.parentsByChild = make(map[string]string)
	e.childrenByParent = make(map[string][]string)
	net, err := utils.LoadNetwork()
	if err != nil {
		log.Fatal(err)
	}
	e.Net = *net

	if len(e.Net.ParentsStations) > 0 && e.Net.ParentsStations[0].Lat == 0 && e.Net.ParentsStations[0].Lon == 0 {
		utils.EnrichParentsStations(&e.Net)
	}

	for i := range e.Net.ParentsStations {
		p := &e.Net.ParentsStations[i]
		p.PID = utils.IdRemoveIDFMPrefix(p.PID)
		for j := range p.ChildrenStops {
			p.ChildrenStops[j] = utils.IdRemoveIDFMPrefix(p.ChildrenStops[j])
		}
		for j := range p.Routes {
			p.Routes[j].RID = utils.IdRemoveIDFMPrefix(p.Routes[j].RID)
		}
	}

	for _, p := range e.Net.ParentsStations {
		cleanPID := utils.IdRemoveIDFMPrefix(p.PID)
		e.childrenByParent[cleanPID] = p.ChildrenStops
		for _, c := range p.ChildrenStops {
			e.parentsByChild[utils.IdRemoveIDFMPrefix(c)] = cleanPID
		}
	}
	for _, s := range e.Net.Stops {
		cleanSID := utils.IdRemoveIDFMPrefix(s.SID)
		cleanPID := utils.IdRemoveIDFMPrefix(s.ParentStation)
		if cleanPID != "" {
			e.parentsByChild[cleanSID] = cleanPID
			e.childrenByParent[cleanPID] = append(e.childrenByParent[cleanPID], cleanSID)
		}
	}

	extra, err := utils.LoadExtra()
	if err == nil && extra != nil {
		e.Extras = *extra
	}

	e.shapesIndex = make(map[string][][2]float64, 15000)
	for _, s := range e.Extras.Shapes {
		e.shapesIndex[s.ShapeID] = append(e.shapesIndex[s.ShapeID], [2]float64{s.Lat, s.Lon})
	}
	for id, pts := range e.shapesIndex {
		clean := utils.IdRemoveIDFMPrefix(id)
		if clean != id {
			e.shapesIndex[clean] = pts
		}
	}

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
	for _, t := range e.Net.Trips {
		if t.ServiceID != "" {
			e.CalendarIndex.GetOrRegisterService(t.ServiceID)
		}
	}

	e.Graph = utils.BuildGraph(net, e.CalendarIndex)
	e.ReverseGraph = utils.BuildReverseGraph(e.Graph)

	e.FetchDisruptions()
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			e.FetchDisruptions()
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
