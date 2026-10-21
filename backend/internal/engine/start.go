package engine

import (
	"log"
	"time"

	"maps/backend/internal/types"
	"maps/backend/internal/utils"
)

type Engine struct {
	Net           types.Network
	Graph         types.GraphType
	ReverseGraph  types.GraphType
	Extras        types.ExtraData
	CalendarIndex *utils.CalendarIndex
}

func StartEngine() Engine {
	var e Engine
	net, err := utils.LoadNetwork()
	if err != nil {
		log.Fatal(err)
	}
	e.Net = *net

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

	return e
}

func (e *Engine) GetActiveServices(target time.Time) []bool {
	if e.CalendarIndex == nil {
		return nil
	}
	return e.CalendarIndex.GetActiveServices(target)
}
