package engine

import (
	"cmp"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"maps/backend/internal/env"
	"maps/backend/internal/types"
	"maps/backend/internal/utils"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"
)

var delaysHttpClient = &http.Client{
	Timeout: 25 * time.Second,
}

type RealtimeCall struct {
	AimedDepSec    uint32
	ExpectedDepSec uint32
	AimedArrSec    uint32
	ExpectedArrSec uint32
	DepDelaySec    int
	ArrDelaySec    int
	HasDep         bool
	HasArr         bool
	Cancelled      bool
}

type tripStopKey struct {
	TripIndex uint32
	CleanSID  string
}

type routeStopKey struct {
	CleanRID string
	CleanSID string
}

type DelaysData struct {
	UpdatedAt     time.Time
	ByTripAndStop map[tripStopKey]RealtimeCall
	ByTripDelay   map[uint32]int
	ByRouteStop   map[routeStopKey][]RealtimeCall
}

type ActiveDelays struct {
	engine *Engine
	data   *DelaysData
}

type RealtimeSchedule struct {
	Sched       types.DepartureTime
	RealDep     uint32
	RealArr     uint32
	DepDelaySec int
	ArrDelaySec int
}

type siriEstimatedTimetableResp struct {
	Siri struct {
		ServiceDelivery struct {
			EstimatedTimetableDelivery []struct {
				EstimatedJourneyVersionFrame []struct {
					EstimatedVehicleJourney []struct {
						LineRef struct {
							Value string `json:"value"`
						} `json:"LineRef"`
						FramedVehicleJourneyRef struct {
							DatedVehicleJourneyRef string `json:"DatedVehicleJourneyRef"`
						} `json:"FramedVehicleJourneyRef"`
						DatedVehicleJourneyRef struct {
							Value string `json:"value"`
						} `json:"DatedVehicleJourneyRef"`
						VehicleJourneyName []struct {
							Value string `json:"value"`
						} `json:"VehicleJourneyName"`
						EstimatedCalls struct {
							EstimatedCall []struct {
								StopPointRef struct {
									Value string `json:"value"`
								} `json:"StopPointRef"`
								ExpectedArrivalTime   string `json:"ExpectedArrivalTime"`
								AimedArrivalTime      string `json:"AimedArrivalTime"`
								ExpectedDepartureTime string `json:"ExpectedDepartureTime"`
								AimedDepartureTime    string `json:"AimedDepartureTime"`
								DepartureStatus       string `json:"DepartureStatus"`
								ArrivalStatus         string `json:"ArrivalStatus"`
							} `json:"EstimatedCall"`
						} `json:"EstimatedCalls"`
					} `json:"EstimatedVehicleJourney"`
				} `json:"EstimatedJourneyVersionFrame"`
			} `json:"EstimatedTimetableDelivery"`
		} `json:"ServiceDelivery"`
	} `json:"Siri"`
}

func extractCleanSiriID(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, ":LOC")
	raw = strings.TrimSuffix(raw, ":")
	if idx := strings.LastIndex(raw, ":"); idx != -1 {
		return raw[idx+1:]
	}
	return raw
}

func parseSiriTimeSec(raw string, baseDate time.Time) (uint32, bool) {
	if raw == "" {
		return 0, false
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return 0, false
		}
	}
	tParis := t.In(parisLoc)
	if !baseDate.IsZero() {
		diffSec := int(tParis.Sub(baseDate).Seconds())
		if diffSec >= 0 && diffSec < 48*3600 {
			return uint32(diffSec), true
		}
	}
	sec := tParis.Hour()*3600 + tParis.Minute()*60 + tParis.Second()
	if sec < 0 {
		return 0, false
	}
	return uint32(sec), true
}

func (e *Engine) buildTripLookupIndices() {
	e.tripIdxByUUID = make(map[string][]uint32, len(e.Net.Trips))
	e.tripIdxByRouteSName = make(map[string][]uint32, len(e.Net.Trips)/2)

	for i, t := range e.Net.Trips {
		tripIdx := uint32(i + 1)
		tid := strings.TrimSpace(t.TID)
		if tid != "" {
			cleanTID := extractCleanSiriID(tid)
			if cleanTID != "" {
				e.tripIdxByUUID[cleanTID] = append(e.tripIdxByUUID[cleanTID], tripIdx)
			}
		}
		sname := strings.TrimSpace(t.SName)
		if sname != "" {
			cleanRID := utils.IdRemoveIDFMPrefix(t.RID)
			key := fmt.Sprintf("%s:%s", cleanRID, sname)
			e.tripIdxByRouteSName[key] = append(e.tripIdxByRouteSName[key], tripIdx)
		}
	}
}

func (e *Engine) FetchDelays() {
	apiKey := env.GetEnv("PRIM_API_KEY")
	if apiKey == "" {
		return
	}

	url := "https://prim.iledefrance-mobilites.fr/marketplace/estimated-timetable?LineRef=ALL"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}
	req.Header.Set("apiKey", apiKey)
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := delaysHttpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return
	}

	var reader io.Reader = resp.Body
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return
		}
		defer gz.Close()
		reader = gz
	}

	var payload siriEstimatedTimetableResp
	if err := json.NewDecoder(reader).Decode(&payload); err != nil {
		return
	}

	nowParis := utils.NowInParis()
	y, m, d := nowParis.Date()
	baseDate := time.Date(y, m, d, 0, 0, 0, 0, parisLoc)

	byTripAndStop := make(map[tripStopKey]RealtimeCall, 32768)
	byTripDelay := make(map[uint32]int, 4096)
	byRouteStop := make(map[routeStopKey][]RealtimeCall, 16384)

	for _, deliv := range payload.Siri.ServiceDelivery.EstimatedTimetableDelivery {
		for _, frame := range deliv.EstimatedJourneyVersionFrame {
			for _, vj := range frame.EstimatedVehicleJourney {
				cleanRID := extractCleanSiriID(vj.LineRef.Value)
				if cleanRID == "" {
					continue
				}

				var matchedTripIndices []uint32
				rawJourneyRef := vj.DatedVehicleJourneyRef.Value
				if rawJourneyRef == "" {
					rawJourneyRef = vj.FramedVehicleJourneyRef.DatedVehicleJourneyRef
				}
				if rawJourneyRef != "" && e.tripIdxByUUID != nil {
					cleanRef := extractCleanSiriID(rawJourneyRef)
					if indices, ok := e.tripIdxByUUID[cleanRef]; ok {
						matchedTripIndices = append(matchedTripIndices, indices...)
					}
				}
				if len(matchedTripIndices) == 0 && len(vj.VehicleJourneyName) > 0 && e.tripIdxByRouteSName != nil {
					vName := strings.TrimSpace(vj.VehicleJourneyName[0].Value)
					if vName != "" {
						if indices, ok := e.tripIdxByRouteSName[fmt.Sprintf("%s:%s", cleanRID, vName)]; ok {
							matchedTripIndices = append(matchedTripIndices, indices...)
						}
					}
				}

				var latestDelay int
				hasLatestDelay := false

				for _, c := range vj.EstimatedCalls.EstimatedCall {
					cleanSID := extractCleanSiriID(c.StopPointRef.Value)
					if cleanSID == "" {
						continue
					}

					aimedDep, hasAimedDep := parseSiriTimeSec(c.AimedDepartureTime, baseDate)
					expDep, hasExpDep := parseSiriTimeSec(c.ExpectedDepartureTime, baseDate)
					aimedArr, hasAimedArr := parseSiriTimeSec(c.AimedArrivalTime, baseDate)
					expArr, hasExpArr := parseSiriTimeSec(c.ExpectedArrivalTime, baseDate)

					if !hasAimedDep && hasAimedArr {
						aimedDep = aimedArr
						hasAimedDep = true
					}
					if !hasExpDep && hasExpArr {
						expDep = expArr
						hasExpDep = true
					}
					if !hasAimedArr && hasAimedDep {
						aimedArr = aimedDep
						hasAimedArr = true
					}
					if !hasExpArr && hasExpDep {
						expArr = expDep
						hasExpArr = true
					}

					if !hasExpDep && !hasExpArr {
						continue
					}

					if !hasAimedDep {
						aimedDep = expDep
						hasAimedDep = true
					}
					if !hasAimedArr {
						aimedArr = expArr
						hasAimedArr = true
					}

					depDelay := int(expDep) - int(aimedDep)
					arrDelay := int(expArr) - int(aimedArr)

					cancelled := strings.EqualFold(c.DepartureStatus, "CANCELLED") ||
						strings.EqualFold(c.ArrivalStatus, "CANCELLED")

					rc := RealtimeCall{
						AimedDepSec:    aimedDep,
						ExpectedDepSec: expDep,
						AimedArrSec:    aimedArr,
						ExpectedArrSec: expArr,
						DepDelaySec:    depDelay,
						ArrDelaySec:    arrDelay,
						HasDep:         hasExpDep,
						HasArr:         hasExpArr,
						Cancelled:      cancelled,
					}

					latestDelay = depDelay
					hasLatestDelay = true

					stopIDs := []string{cleanSID}
					if e.parentsByChild != nil {
						if parentID, ok := e.parentsByChild[cleanSID]; ok && parentID != "" && parentID != cleanSID {
							stopIDs = append(stopIDs, parentID)
						}
					}

					for _, sid := range stopIDs {
						rsKey := routeStopKey{CleanRID: cleanRID, CleanSID: sid}
						byRouteStop[rsKey] = append(byRouteStop[rsKey], rc)

						for _, tIdx := range matchedTripIndices {
							byTripAndStop[tripStopKey{TripIndex: tIdx, CleanSID: sid}] = rc
						}
					}
				}

				if hasLatestDelay {
					for _, tIdx := range matchedTripIndices {
						byTripDelay[tIdx] = latestDelay
					}
				}
			}
		}
	}

	data := &DelaysData{
		UpdatedAt:     nowParis,
		ByTripAndStop: byTripAndStop,
		ByTripDelay:   byTripDelay,
		ByRouteStop:   byRouteStop,
	}

	if e.delaysMu != nil {
		e.delaysMu.Lock()
		e.delays = data
		e.delaysMu.Unlock()
	} else {
		e.delays = data
	}
}

func (e *Engine) GetActiveDelays(targetTime time.Time) *ActiveDelays {
	var data *DelaysData
	if e.delaysMu != nil {
		e.delaysMu.RLock()
		data = e.delays
		e.delaysMu.RUnlock()
	} else {
		data = e.delays
	}
	if data == nil {
		return nil
	}

	now := utils.NowInParis()
	targetParis := utils.InParis(targetTime)
	diff := targetParis.Sub(now)
	if diff < -3*time.Hour || diff > 6*time.Hour {
		return nil
	}

	return &ActiveDelays{
		engine: e,
		data:   data,
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func applyDelayToSec(sec uint32, delay int) uint32 {
	res := int(sec) + delay
	if res < 0 {
		return 0
	}
	return uint32(res)
}

func (ad *ActiveDelays) findCallForStop(tripIndex uint32, cleanRID, cleanSID string, schedTime uint32, isArrival bool) (RealtimeCall, bool) {
	if ad == nil || ad.data == nil || cleanSID == "" {
		return RealtimeCall{}, false
	}

	candidates := []string{cleanSID}
	if ad.engine != nil && ad.engine.parentsByChild != nil {
		if p, ok := ad.engine.parentsByChild[cleanSID]; ok && p != "" && p != cleanSID {
			candidates = append(candidates, p)
		}
	}

	if tripIndex > 0 {
		for _, sid := range candidates {
			if rc, ok := ad.data.ByTripAndStop[tripStopKey{TripIndex: tripIndex, CleanSID: sid}]; ok {
				refAimed := rc.AimedDepSec
				if isArrival && rc.HasArr {
					refAimed = rc.AimedArrSec
				}
				if absInt(int(refAimed)-int(schedTime)) <= 1800 {
					return rc, true
				}
			}
		}
	}

	if cleanRID != "" {
		bestDiff := 121
		var bestCall RealtimeCall
		found := false

		for _, sid := range candidates {
			calls := ad.data.ByRouteStop[routeStopKey{CleanRID: cleanRID, CleanSID: sid}]
			for _, rc := range calls {
				refAimed := rc.AimedDepSec
				if isArrival && rc.HasArr {
					refAimed = rc.AimedArrSec
				}
				diff := absInt(int(refAimed) - int(schedTime))
				if diff < bestDiff {
					bestDiff = diff
					bestCall = rc
					found = true
				}
			}
		}
		if found {
			return bestCall, true
		}
	}

	if tripIndex > 0 {
		if dSec, ok := ad.data.ByTripDelay[tripIndex]; ok {
			return RealtimeCall{
				AimedDepSec:    schedTime,
				ExpectedDepSec: applyDelayToSec(schedTime, dSec),
				AimedArrSec:    schedTime,
				ExpectedArrSec: applyDelayToSec(schedTime, dSec),
				DepDelaySec:    dSec,
				ArrDelaySec:    dSec,
				HasDep:         true,
				HasArr:         true,
			}, true
		}
	}

	return RealtimeCall{}, false
}

func (ad *ActiveDelays) ResolveScheduleRealtime(rid, fromSID, toSID string, sched types.DepartureTime) (realDep, realArr uint32, depDelay, arrDelay int, cancelled bool) {
	if ad == nil || ad.data == nil {
		return sched.Dep, sched.Arr, 0, 0, false
	}

	cleanRID := utils.IdRemoveIDFMPrefix(rid)
	cleanFrom := utils.IdRemoveIDFMPrefix(fromSID)
	cleanTo := utils.IdRemoveIDFMPrefix(toSID)

	depCall, okDep := ad.findCallForStop(sched.TripIndex, cleanRID, cleanFrom, sched.Dep, false)
	if okDep && depCall.Cancelled {
		return 0, 0, 0, 0, true
	}

	arrCall, okArr := ad.findCallForStop(sched.TripIndex, cleanRID, cleanTo, sched.Arr, true)
	if okArr && arrCall.Cancelled {
		return 0, 0, 0, 0, true
	}

	if okDep {
		depDelay = depCall.DepDelaySec
		realDep = applyDelayToSec(sched.Dep, depDelay)
	} else if okArr {
		depDelay = arrCall.ArrDelaySec
		realDep = applyDelayToSec(sched.Dep, depDelay)
	} else {
		realDep = sched.Dep
	}

	if okArr {
		arrDelay = arrCall.ArrDelaySec
		realArr = applyDelayToSec(sched.Arr, arrDelay)
	} else if okDep {
		arrDelay = depDelay
		realArr = applyDelayToSec(sched.Arr, arrDelay)
	} else {
		realArr = sched.Arr
	}

	if realArr < realDep {
		schedRide := uint32(0)
		if sched.Arr > sched.Dep {
			schedRide = sched.Arr - sched.Dep
		}
		realArr = realDep + schedRide
		arrDelay = int(realArr) - int(sched.Arr)
	}

	return realDep, realArr, depDelay, arrDelay, false
}

func (e *Engine) FindNextDepartureRealtime(edge *types.Edge, currSec uint32, activeServices []bool, ad *ActiveDelays) (RealtimeSchedule, bool) {
	list := e.FindNextDeparturesRealtime(edge, currSec, 1, 6*3600, activeServices, ad)
	if len(list) == 0 {
		return RealtimeSchedule{}, false
	}
	return list[0], true
}

func (e *Engine) FindNextDeparturesRealtime(edge *types.Edge, currSec uint32, limit int, maxWindowSec uint32, activeServices []bool, ad *ActiveDelays) []RealtimeSchedule {
	if edge == nil || len(edge.Schedules) == 0 || limit <= 0 {
		return nil
	}

	if ad == nil {
		scheds := utils.FindNextDepartures(edge.Schedules, currSec, limit, maxWindowSec, activeServices)
		if len(scheds) == 0 {
			if s, ok := utils.FindNextDeparture(edge.Schedules, currSec, activeServices); ok && limit == 1 {
				scheds = []types.DepartureTime{s}
			}
		}
		res := make([]RealtimeSchedule, 0, len(scheds))
		for _, s := range scheds {
			res = append(res, RealtimeSchedule{
				Sched:   s,
				RealDep: s.Dep,
				RealArr: s.Arr,
			})
		}
		return res
	}

	lookbackSec := uint32(0)
	if currSec > 3600 {
		lookbackSec = currSec - 3600
	}
	idx, _ := slices.BinarySearchFunc(edge.Schedules, lookbackSec, func(s types.DepartureTime, target uint32) int {
		return cmp.Compare(s.Dep, target)
	})

	maxDep := currSec + maxWindowSec
	var candidates []RealtimeSchedule

	for i := idx; i < len(edge.Schedules); i++ {
		s := edge.Schedules[i]
		if s.Dep > maxDep+3600 {
			break
		}
		if len(activeServices) > 0 && (int(s.ServiceID) >= len(activeServices) || !activeServices[s.ServiceID]) {
			continue
		}

		realDep, realArr, depDelay, arrDelay, cancelled := ad.ResolveScheduleRealtime(edge.RID, edge.FromSID, edge.ToSID, s)
		if cancelled {
			continue
		}
		if realDep < currSec {
			continue
		}
		if realDep > maxDep && len(candidates) >= limit {
			break
		}

		candidates = append(candidates, RealtimeSchedule{
			Sched:       s,
			RealDep:     realDep,
			RealArr:     realArr,
			DepDelaySec: depDelay,
			ArrDelaySec: arrDelay,
		})

		if s.Dep > currSec && len(candidates) >= limit+4 {
			break
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].RealDep < candidates[j].RealDep
	})

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

func (e *Engine) FindTripDepartureRealtime(edge *types.Edge, tripIndex uint32, currSec uint32, activeServices []bool, ad *ActiveDelays) (RealtimeSchedule, bool) {
	if edge == nil {
		return RealtimeSchedule{}, false
	}
	for i := range edge.Schedules {
		s := edge.Schedules[i]
		if s.TripIndex == tripIndex {
			if len(activeServices) > 0 && (int(s.ServiceID) >= len(activeServices) || !activeServices[s.ServiceID]) {
				return RealtimeSchedule{}, false
			}
			realDep, realArr, depDelay, arrDelay, cancelled := ad.ResolveScheduleRealtime(edge.RID, edge.FromSID, edge.ToSID, s)
			if cancelled {
				return RealtimeSchedule{}, false
			}
			if realDep >= currSec {
				return RealtimeSchedule{
					Sched:       s,
					RealDep:     realDep,
					RealArr:     realArr,
					DepDelaySec: depDelay,
					ArrDelaySec: arrDelay,
				}, true
			}

			if currSec > realDep && currSec-realDep <= 90 {
				adjDep := currSec
				adjArr := realArr
				if adjArr < adjDep {
					adjArr = adjDep + (s.Arr - s.Dep)
				}
				return RealtimeSchedule{
					Sched:       s,
					RealDep:     adjDep,
					RealArr:     adjArr,
					DepDelaySec: int(adjDep) - int(s.Dep),
					ArrDelaySec: int(adjArr) - int(s.Arr),
				}, true
			}
			return RealtimeSchedule{}, false
		}
	}
	return RealtimeSchedule{}, false
}

func (e *Engine) FindPrevDeparturesRealtime(edge *types.Edge, currSec uint32, limit int, maxWindowSec uint32, activeServices []bool, ad *ActiveDelays) []RealtimeSchedule {
	if edge == nil || len(edge.Schedules) == 0 || limit <= 0 {
		return nil
	}

	if ad == nil {
		scheds := utils.FindPrevDepartures(edge.Schedules, currSec, limit, maxWindowSec, activeServices)
		res := make([]RealtimeSchedule, 0, len(scheds))
		for _, s := range scheds {
			res = append(res, RealtimeSchedule{
				Sched:   s,
				RealDep: s.Dep,
				RealArr: s.Arr,
			})
		}
		return res
	}

	searchUntil := currSec + 600
	low := 0
	high := len(edge.Schedules) - 1
	bestIdx := -1
	for low <= high {
		mid := low + (high-low)/2
		if edge.Schedules[mid].Arr <= searchUntil {
			bestIdx = mid
			low = mid + 1
		} else {
			high = mid - 1
		}
	}
	if bestIdx == -1 {
		return nil
	}

	minArr := uint32(0)
	if currSec > maxWindowSec+3600 {
		minArr = currSec - maxWindowSec - 3600
	}

	var candidates []RealtimeSchedule
	for i := bestIdx; i >= 0; i-- {
		s := edge.Schedules[i]
		if s.Arr < minArr {
			break
		}
		if len(activeServices) > 0 && (int(s.ServiceID) >= len(activeServices) || !activeServices[s.ServiceID]) {
			continue
		}
		realDep, realArr, depDelay, arrDelay, cancelled := ad.ResolveScheduleRealtime(edge.RID, edge.FromSID, edge.ToSID, s)
		if cancelled {
			continue
		}
		if realArr > currSec {
			continue
		}
		if currSec > maxWindowSec && realArr < currSec-maxWindowSec && len(candidates) >= limit {
			break
		}

		candidates = append(candidates, RealtimeSchedule{
			Sched:       s,
			RealDep:     realDep,
			RealArr:     realArr,
			DepDelaySec: depDelay,
			ArrDelaySec: arrDelay,
		})
		if s.Arr < currSec && len(candidates) >= limit+4 {
			break
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].RealArr > candidates[j].RealArr
	})

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

func (e *Engine) FindTripArrivalRealtime(edge *types.Edge, tripIndex uint32, currSec uint32, activeServices []bool, ad *ActiveDelays) (RealtimeSchedule, bool) {
	if edge == nil {
		return RealtimeSchedule{}, false
	}
	for i := range edge.Schedules {
		s := edge.Schedules[i]
		if s.TripIndex == tripIndex {
			if len(activeServices) > 0 && (int(s.ServiceID) >= len(activeServices) || !activeServices[s.ServiceID]) {
				return RealtimeSchedule{}, false
			}
			realDep, realArr, depDelay, arrDelay, cancelled := ad.ResolveScheduleRealtime(edge.RID, edge.FromSID, edge.ToSID, s)
			if cancelled {
				return RealtimeSchedule{}, false
			}
			if realArr <= currSec {
				return RealtimeSchedule{
					Sched:       s,
					RealDep:     realDep,
					RealArr:     realArr,
					DepDelaySec: depDelay,
					ArrDelaySec: arrDelay,
				}, true
			}
			if realArr > currSec && realArr-currSec <= 90 {
				adjArr := currSec
				adjDep := realDep
				if adjDep > adjArr {
					if adjArr >= (s.Arr - s.Dep) {
						adjDep = adjArr - (s.Arr - s.Dep)
					} else {
						adjDep = 0
					}
				}
				return RealtimeSchedule{
					Sched:       s,
					RealDep:     adjDep,
					RealArr:     adjArr,
					DepDelaySec: int(adjDep) - int(s.Dep),
					ArrDelaySec: int(adjArr) - int(s.Arr),
				}, true
			}
			return RealtimeSchedule{}, false
		}
	}
	return RealtimeSchedule{}, false
}
