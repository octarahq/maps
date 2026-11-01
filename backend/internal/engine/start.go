package engine

import (
	"log"
	"sync"
	"time"

	"maps/backend/internal/types"
	"maps/backend/internal/utils"
)

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
