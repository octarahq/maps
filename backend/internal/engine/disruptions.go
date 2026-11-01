package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"maps/backend/internal/env"
	"maps/backend/internal/types"
	"maps/backend/internal/utils"
	"net/http"
	"time"
)

var (
	parisLoc = func() *time.Location {
		loc, err := time.LoadLocation("Europe/Paris")
		if err != nil {
			return time.Local
		}
		return loc
	}()

	primClient = &http.Client{
		Timeout: 15 * time.Second,
	}
)

type DepartureRes struct {
	Id                 string `json:"id"`
	ApplicationPeriods []struct {
		Begin string `json:"begin"`
		End   string `json:"end"`
	} `json:"applicationPeriods"`
	LastUpdate       string `json:"lastUpdate"`
	Cause            string `json:"cause"`
	Severity         string `json:"severity"`
	Title            string `json:"title"`
	Message          string `json:"message"`
	ShortMessage     string `json:"shortMessage"`
	ImpactedSections []struct {
		LineID string `json:"lineId"`
		From   struct {
			Id string `json:"id"`
		} `json:"from"`
		To struct {
			Id string `json:"id"`
		} `json:"to"`
	} `json:"impactedSections"`
}

func (e *Engine) FetchDisruptions() {
	url := "https://prim.iledefrance-mobilites.fr/marketplace/disruptions_bulk/disruptions/v2"
	rData, err := fetchData(url)
	if err != nil {
		return
	}
	defer rData.Close()

	var jData struct {
		Disruptions []DepartureRes `json:"disruptions"`
	}

	if err := json.NewDecoder(rData).Decode(&jData); err != nil {
		return
	}

	departures := parseDisruptions(jData)

	if e.disruptionsMu != nil {
		e.disruptionsMu.Lock()
		e.Disruptions = departures
		e.disruptionsMu.Unlock()
	} else {
		e.Disruptions = departures
	}
}

func parseIDFMTime(t string) time.Time {
	parsed, err := time.ParseInLocation("20060102T150405", t, parisLoc)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func parseDisruptions(disruptions struct {
	Disruptions []DepartureRes `json:"disruptions"`
}) []types.Disruption {
	res := make([]types.Disruption, 0, len(disruptions.Disruptions))

	for _, d := range disruptions.Disruptions {
		applicationsPeriods := make([]types.DisruptionPeriod, 0, len(d.ApplicationPeriods))
		for _, a := range d.ApplicationPeriods {
			applicationsPeriods = append(applicationsPeriods, types.DisruptionPeriod{
				Begin: parseIDFMTime(a.Begin),
				End:   parseIDFMTime(a.End),
			})
		}

		sections := make([]types.DisruptionImpactedSections, 0, len(d.ImpactedSections))
		for _, s := range d.ImpactedSections {
			sections = append(sections, types.DisruptionImpactedSections{
				RID:  utils.IdRemoveIDFMPrefix(s.LineID),
				From: utils.IdRemoveIDFMPrefix(s.From.Id),
				To:   utils.IdRemoveIDFMPrefix(s.To.Id),
			})
		}

		cause := types.DisruptionCauseUnknow
		switch d.Cause {
		case "TRAVAUX":
			cause = types.DisruptionCauseTravaux
		case "PERTURBATION":
			cause = types.DisruptionCauseDisturbance
		case "INFORMATION":
			cause = types.DisruptionCauseInfo
		}

		severity := types.DisruptionSeverityUnknow
		switch d.Severity {
		case "BLOQUANTE":
			severity = types.DisruptionSeverityBloqued
		case "PERTURBEE":
			severity = types.DisruptionSeverityPerturbate
		case "INFORMATION":
			severity = types.DisruptionSeverityInfo
		}

		disruption := types.Disruption{
			DID:              d.Id,
			Title:            d.Title,
			Message:          d.Message,
			SMessage:         d.ShortMessage,
			Periods:          applicationsPeriods,
			ImpactedSections: sections,
			Cause:            cause,
			Severity:         severity,
		}

		res = append(res, disruption)
	}

	return res
}

func fetchData(url string) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apiKey", env.GetEnv("PRIM_API_KEY"))

	resp, err := primClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		resp.Body.Close()
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	return resp.Body, nil
}

type ActiveDisruptions struct {
	BlockedStops     map[string]bool
	BlockedLineStops map[string]bool
	BlockedSections  map[string]bool
	PerturbedLines   map[string]bool
	PerturbedStops   map[string]bool
}

func (e *Engine) GetActiveDisruptions(targetTime time.Time) *ActiveDisruptions {
	if targetTime.IsZero() {
		targetTime = time.Now()
	}

	ad := &ActiveDisruptions{
		BlockedStops:     make(map[string]bool),
		BlockedLineStops: make(map[string]bool),
		BlockedSections:  make(map[string]bool),
		PerturbedLines:   make(map[string]bool),
		PerturbedStops:   make(map[string]bool),
	}

	var disruptions []types.Disruption
	if e.disruptionsMu != nil {
		e.disruptionsMu.RLock()
		disruptions = e.Disruptions
		e.disruptionsMu.RUnlock()
	} else {
		disruptions = e.Disruptions
	}

	markStopBlocked := func(id string) {
		cleanID := utils.IdRemoveIDFMPrefix(id)
		if cleanID == "" {
			return
		}
		ad.BlockedStops[cleanID] = true
		if e.childrenByParent != nil {
			if children, ok := e.childrenByParent[cleanID]; ok {
				for _, c := range children {
					ad.BlockedStops[utils.IdRemoveIDFMPrefix(c)] = true
				}
			}
		}
		if e.parentsByChild != nil {
			if parent, ok := e.parentsByChild[cleanID]; ok {
				ad.BlockedStops[parent] = true
			}
		}
	}

	markStopBlockedOnLine := func(id string, rid string) {
		cleanID := utils.IdRemoveIDFMPrefix(id)
		cleanRID := utils.IdRemoveIDFMPrefix(rid)
		if cleanID == "" || cleanRID == "" {
			return
		}
		ad.BlockedLineStops[fmt.Sprintf("%s:%s", cleanRID, cleanID)] = true
		if e.childrenByParent != nil {
			if children, ok := e.childrenByParent[cleanID]; ok {
				for _, c := range children {
					ad.BlockedLineStops[fmt.Sprintf("%s:%s", cleanRID, utils.IdRemoveIDFMPrefix(c))] = true
				}
			}
		}
		if e.parentsByChild != nil {
			if parent, ok := e.parentsByChild[cleanID]; ok {
				ad.BlockedLineStops[fmt.Sprintf("%s:%s", cleanRID, parent)] = true
			}
		}
	}

	for _, d := range disruptions {
		if len(d.Periods) > 0 {
			active := false
			for _, p := range d.Periods {
				if (p.Begin.IsZero() || !targetTime.Before(p.Begin)) && (p.End.IsZero() || !targetTime.After(p.End)) {
					active = true
					break
				}
			}
			if !active {
				continue
			}
		}

		if d.Severity == types.DisruptionSeverityBloqued {
			for _, s := range d.ImpactedSections {
				rid := utils.IdRemoveIDFMPrefix(s.RID)
				from := utils.IdRemoveIDFMPrefix(s.From)
				to := utils.IdRemoveIDFMPrefix(s.To)

				if rid == "" {
					if from != "" {
						markStopBlocked(from)
					}
					if to != "" {
						markStopBlocked(to)
					}
				} else {
					if from != "" {
						markStopBlockedOnLine(from, rid)
					}
					if to != "" {
						markStopBlockedOnLine(to, rid)
					}
					if from != "" && to != "" && from != to {
						ad.BlockedSections[fmt.Sprintf("%s:%s->%s", rid, from, to)] = true
						ad.BlockedSections[fmt.Sprintf("%s:%s->%s", rid, to, from)] = true
					}
				}
			}
		} else if d.Severity == types.DisruptionSeverityPerturbate || d.Cause == types.DisruptionCauseDisturbance {
			for _, s := range d.ImpactedSections {
				rid := utils.IdRemoveIDFMPrefix(s.RID)
				from := utils.IdRemoveIDFMPrefix(s.From)
				to := utils.IdRemoveIDFMPrefix(s.To)

				if rid != "" {
					ad.PerturbedLines[rid] = true
				}
				if from != "" {
					ad.PerturbedStops[from] = true
					if e.parentsByChild != nil {
						if parent, ok := e.parentsByChild[from]; ok {
							ad.PerturbedStops[parent] = true
						}
					}
				}
				if to != "" {
					ad.PerturbedStops[to] = true
					if e.parentsByChild != nil {
						if parent, ok := e.parentsByChild[to]; ok {
							ad.PerturbedStops[parent] = true
						}
					}
				}
			}
		}
	}

	return ad
}

func (ad *ActiveDisruptions) IsStopBlocked(sid string) bool {
	if ad == nil || len(ad.BlockedStops) == 0 {
		return false
	}
	clean := utils.IdRemoveIDFMPrefix(sid)
	return ad.BlockedStops[clean]
}

func (ad *ActiveDisruptions) IsStopBlockedOnLine(sid string, rid string) bool {
	if ad == nil {
		return false
	}
	cleanSID := utils.IdRemoveIDFMPrefix(sid)
	if ad.BlockedStops[cleanSID] {
		return true
	}
	if len(ad.BlockedLineStops) == 0 || rid == "" {
		return false
	}
	cleanRID := utils.IdRemoveIDFMPrefix(rid)
	return ad.BlockedLineStops[fmt.Sprintf("%s:%s", cleanRID, cleanSID)]
}

func (ad *ActiveDisruptions) IsEdgeBlocked(edge *types.Edge) bool {
	if ad == nil || edge == nil || edge.RID == "" || len(ad.BlockedSections) == 0 {
		return false
	}
	rid := utils.IdRemoveIDFMPrefix(edge.RID)
	from := utils.IdRemoveIDFMPrefix(edge.FromSID)
	to := utils.IdRemoveIDFMPrefix(edge.ToSID)
	if ad.BlockedSections[fmt.Sprintf("%s:%s->%s", rid, from, to)] {
		return true
	}
	return false
}

func (ad *ActiveDisruptions) IsEdgePerturbed(edge *types.Edge) bool {
	if ad == nil || edge == nil || edge.RID == "" {
		return false
	}
	rid := utils.IdRemoveIDFMPrefix(edge.RID)
	if ad.PerturbedLines[rid] {
		return true
	}
	if len(ad.PerturbedStops) > 0 {
		from := utils.IdRemoveIDFMPrefix(edge.FromSID)
		to := utils.IdRemoveIDFMPrefix(edge.ToSID)
		if ad.PerturbedStops[from] || ad.PerturbedStops[to] {
			return true
		}
	}
	return false
}

func (e *Engine) IsStopBlockedAt(sid string, targetTime time.Time) bool {
	ad := e.GetActiveDisruptions(targetTime)
	return ad.IsStopBlocked(sid)
}
