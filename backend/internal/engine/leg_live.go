package engine

import (
	"encoding/json"
	"fmt"
	"maps/backend/internal/env"
	"maps/backend/internal/utils"
	"net/http"
	"strings"
	"sync"
	"time"
)

const stopMonitoringCacheTTL = 30 * time.Second

type stopMonCall struct {
	CleanRID       string
	TripUUID       string
	VehicleName    string
	Headsign       string
	AimedDepSec    uint32
	ExpectedDepSec uint32
	AimedArrSec    uint32
	ExpectedArrSec uint32
	DepDelaySec    int
	ArrDelaySec    int
	HasDep         bool
	HasArr         bool
	Cancelled      bool
	DepPlatform    string
	ArrPlatform    string
	IsLongTrain    *bool
}

type stopMonCacheEntry struct {
	fetchedAt time.Time
	calls     []stopMonCall
}

type stopMonitoringCache struct {
	mu      sync.Mutex
	entries map[string]stopMonCacheEntry
}

func newStopMonitoringCache() *stopMonitoringCache {
	return &stopMonitoringCache{
		entries: make(map[string]stopMonCacheEntry),
	}
}

var fetchStopMonitoringRawFn = defaultFetchStopMonitoringRaw

func defaultFetchStopMonitoringRaw(stopAreaID string) ([]byte, error) {
	apiKey := env.GetEnv("PRIM_API_KEY")
	if apiKey == "" || stopAreaID == "" {
		return nil, nil
	}
	monitoringRef := stopAreaID
	if !strings.HasPrefix(monitoringRef, "STIF:") {
		monitoringRef = fmt.Sprintf("STIF:StopArea:SP:%s:", stopAreaID)
	}
	url := fmt.Sprintf("https://prim.iledefrance-mobilites.fr/marketplace/stop-monitoring?MonitoringRef=%s", monitoringRef)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apiKey", apiKey)

	resp, err := stopMonHttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("prim stop-monitoring status %d", resp.StatusCode)
	}
	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

type siriTextOrValue string

func (s *siriTextOrValue) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = siriTextOrValue(str)
		return nil
	}
	var obj struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(data, &obj); err == nil {
		*s = siriTextOrValue(obj.Value)
		return nil
	}
	return nil
}

type siriStopMonitoringResp struct {
	Siri struct {
		ServiceDelivery struct {
			StopMonitoringDelivery []struct {
				MonitoredStopVisit []struct {
					MonitoredVehicleJourney struct {
						LineRef struct {
							Value string `json:"value"`
						} `json:"LineRef"`
						FramedVehicleJourneyRef struct {
							DatedVehicleJourneyRef string `json:"DatedVehicleJourneyRef"`
						} `json:"FramedVehicleJourneyRef"`
						VehicleJourneyName []struct {
							Value string `json:"value"`
						} `json:"VehicleJourneyName"`
						DestinationName []struct {
							Value string `json:"value"`
						} `json:"DestinationName"`
						DirectionName []struct {
							Value string `json:"value"`
						} `json:"DirectionName"`
						VehicleFeatureRef []siriTextOrValue `json:"VehicleFeatureRef"`
						MonitoredCall     struct {
							AimedArrivalTime      string          `json:"AimedArrivalTime"`
							ExpectedArrivalTime   string          `json:"ExpectedArrivalTime"`
							AimedDepartureTime    string          `json:"AimedDepartureTime"`
							ExpectedDepartureTime string          `json:"ExpectedDepartureTime"`
							DepartureStatus       string          `json:"DepartureStatus"`
							ArrivalStatus         string          `json:"ArrivalStatus"`
							DeparturePlatformName siriTextOrValue `json:"DeparturePlatformName"`
							ArrivalPlatformName   siriTextOrValue `json:"ArrivalPlatformName"`
						} `json:"MonitoredCall"`
					} `json:"MonitoredVehicleJourney"`
				} `json:"MonitoredStopVisit"`
			} `json:"StopMonitoringDelivery"`
		} `json:"ServiceDelivery"`
	} `json:"Siri"`
}

func parseStopMonitoringJSON(raw []byte, baseDate time.Time) ([]stopMonCall, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var payload siriStopMonitoringResp
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	var out []stopMonCall
	for _, deliv := range payload.Siri.ServiceDelivery.StopMonitoringDelivery {
		for _, visit := range deliv.MonitoredStopVisit {
			mvj := visit.MonitoredVehicleJourney
			mc := mvj.MonitoredCall
			cleanRID := extractCleanSiriID(mvj.LineRef.Value)
			if cleanRID == "" {
				continue
			}
			tripUUID := extractCleanSiriID(mvj.FramedVehicleJourneyRef.DatedVehicleJourneyRef)
			vName := ""
			if len(mvj.VehicleJourneyName) > 0 {
				vName = strings.TrimSpace(mvj.VehicleJourneyName[0].Value)
			}
			destName := ""
			if len(mvj.DestinationName) > 0 {
				destName = strings.TrimSpace(mvj.DestinationName[0].Value)
			} else if len(mvj.DirectionName) > 0 {
				destName = strings.TrimSpace(mvj.DirectionName[0].Value)
			}
			var isLongTrain *bool
			for _, f := range mvj.VehicleFeatureRef {
				fv := string(f)
				if strings.EqualFold(fv, "longTrain") {
					v := true
					isLongTrain = &v
				} else if strings.EqualFold(fv, "shortTrain") {
					v := false
					isLongTrain = &v
				}
			}

			aimedDep, hasAimedDep := parseSiriTimeSec(mc.AimedDepartureTime, baseDate)
			expDep, hasExpDep := parseSiriTimeSec(mc.ExpectedDepartureTime, baseDate)
			aimedArr, hasAimedArr := parseSiriTimeSec(mc.AimedArrivalTime, baseDate)
			expArr, hasExpArr := parseSiriTimeSec(mc.ExpectedArrivalTime, baseDate)

			if !hasExpDep && hasAimedDep {
				expDep, hasExpDep = aimedDep, true
			}
			if !hasExpArr && hasAimedArr {
				expArr, hasExpArr = aimedArr, true
			}
			if !hasAimedDep && hasExpDep {
				aimedDep, hasAimedDep = expDep, true
			}
			if !hasAimedArr && hasExpArr {
				aimedArr, hasAimedArr = expArr, true
			}

			depPlat := cleanPlatformName(string(mc.DeparturePlatformName))
			arrPlat := cleanPlatformName(string(mc.ArrivalPlatformName))
			if depPlat == "" {
				depPlat = arrPlat
			}
			if arrPlat == "" {
				arrPlat = depPlat
			}

			out = append(out, stopMonCall{
				CleanRID:       cleanRID,
				TripUUID:       tripUUID,
				VehicleName:    vName,
				Headsign:       destName,
				AimedDepSec:    aimedDep,
				ExpectedDepSec: expDep,
				AimedArrSec:    aimedArr,
				ExpectedArrSec: expArr,
				DepDelaySec:    int(expDep) - int(aimedDep),
				ArrDelaySec:    int(expArr) - int(aimedArr),
				HasDep:         hasExpDep,
				HasArr:         hasExpArr,
				Cancelled: strings.EqualFold(mc.DepartureStatus, "CANCELLED") ||
					strings.EqualFold(mc.ArrivalStatus, "CANCELLED"),
				DepPlatform: depPlat,
				ArrPlatform: arrPlat,
				IsLongTrain: isLongTrain,
			})
		}
	}
	return out, nil
}

func (e *Engine) resolveStopAreaID(cleanSID string) string {
	clean := utils.IdRemoveIDFMPrefix(cleanSID)
	if e != nil && e.parentsByChild != nil {
		if p, ok := e.parentsByChild[clean]; ok && p != "" {
			return utils.IdRemoveIDFMPrefix(p)
		}
	}
	return clean
}

func (e *Engine) fetchStopMonitoringCalls(cleanSID string, baseDate time.Time) []stopMonCall {
	stopArea := e.resolveStopAreaID(cleanSID)
	if stopArea == "" {
		return nil
	}
	if e != nil && e.stopMonCache != nil {
		e.stopMonCache.mu.Lock()
		ent, ok := e.stopMonCache.entries[stopArea]
		e.stopMonCache.mu.Unlock()
		if ok && time.Since(ent.fetchedAt) < stopMonitoringCacheTTL {
			return ent.calls
		}
	}
	raw, err := fetchStopMonitoringRawFn(stopArea)
	if err != nil || len(raw) == 0 {
		return nil
	}
	calls, err := parseStopMonitoringJSON(raw, baseDate)
	if err != nil {
		return nil
	}
	if e != nil && e.stopMonCache != nil {
		e.stopMonCache.mu.Lock()
		e.stopMonCache.entries[stopArea] = stopMonCacheEntry{
			fetchedAt: time.Now(),
			calls:     calls,
		}
		e.stopMonCache.mu.Unlock()
	}
	return calls
}

func normMonHeadsign(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	r := strings.NewReplacer("–", " ", "—", " ", "-", " ", "'", " ", "’", " ", "é", "e", "è", "e", "ê", "e", "ë", "e", "à", "a", "â", "a", "î", "i", "ï", "i", "ô", "o", "ù", "u", "û", "u", "ç", "c")
	return strings.Join(strings.Fields(r.Replace(s)), " ")
}

func headsignsOverlap(a, b string) bool {
	na, nb := normMonHeadsign(a), normMonHeadsign(b)
	if na == "" || nb == "" {
		return false
	}
	return na == nb || strings.Contains(na, nb) || strings.Contains(nb, na)
}

func findBestStopMonCall(calls []stopMonCall, cleanRID, tripUUID, trainNum, headsign string, schedSec uint32, isArrival bool) (stopMonCall, bool) {
	if len(calls) == 0 {
		return stopMonCall{}, false
	}
	if tripUUID != "" {
		for _, c := range calls {
			if c.TripUUID == tripUUID {
				return c, true
			}
		}
	}
	if cleanRID != "" && trainNum != "" {
		for _, c := range calls {
			if c.CleanRID == cleanRID && c.VehicleName == trainNum {
				return c, true
			}
		}
	}
	if cleanRID != "" && schedSec > 0 {
		bestScore := 1000
		var best stopMonCall
		found := false
		for _, c := range calls {
			if c.CleanRID != cleanRID {
				continue
			}
			var refSec uint32
			if isArrival && c.HasArr {
				refSec = c.AimedArrSec
			} else if !isArrival && c.HasDep {
				refSec = c.AimedDepSec
			} else if c.HasDep {
				refSec = c.AimedDepSec
			} else if c.HasArr {
				refSec = c.AimedArrSec
			} else {
				continue
			}
			diff := absInt(int(refSec) - int(schedSec))
			if diff > 150 {
				continue
			}
			score := diff
			if headsign != "" && c.Headsign != "" && !headsignsOverlap(headsign, c.Headsign) {
				score += 90
			}
			if score < bestScore {
				bestScore = score
				best = c
				found = true
			}
		}
		if found {
			return best, true
		}
	}
	return stopMonCall{}, false
}

type LegLiveQuery struct {
	RID       string
	FromSID   string
	ToSID     string
	TripIndex uint32
	SchedDep  time.Time
	SchedArr  time.Time
	Headsign  string
}

type LegLiveResult struct {
	Found             bool       `json:"found"`
	Cancelled         bool       `json:"cancelled,omitempty"`
	DepDelaySec       int        `json:"dep_delay_sec"`
	ArrDelaySec       int        `json:"arr_delay_sec"`
	RealDep           *time.Time `json:"real_dep,omitempty"`
	RealArr           *time.Time `json:"real_arr,omitempty"`
	DeparturePlatform string     `json:"departure_platform,omitempty"`
	ArrivalPlatform   string     `json:"arrival_platform,omitempty"`
	IsLongTrain       *bool      `json:"is_long_train,omitempty"`
	Source            string     `json:"source,omitempty"`
}

func (e *Engine) ResolveLegLive(q LegLiveQuery) LegLiveResult {
	cleanRID := utils.IdRemoveIDFMPrefix(q.RID)
	cleanFrom := utils.IdRemoveIDFMPrefix(q.FromSID)
	cleanTo := utils.IdRemoveIDFMPrefix(q.ToSID)

	nowParis := utils.NowInParis()
	y, m, d := nowParis.Date()
	baseDate := time.Date(y, m, d, 0, 0, 0, 0, parisLoc)
	if !q.SchedDep.IsZero() {
		diff := q.SchedDep.In(parisLoc).Sub(baseDate)
		if diff < 0 || diff >= 40*time.Hour {
			ty, tm, td := q.SchedDep.In(parisLoc).Date()
			baseDate = time.Date(ty, tm, td, 0, 0, 0, 0, parisLoc)
		}
	}

	var schedDepSec, schedArrSec uint32
	if !q.SchedDep.IsZero() {
		if s := int(q.SchedDep.In(parisLoc).Sub(baseDate).Seconds()); s >= 0 {
			schedDepSec = uint32(s)
		}
	}
	if !q.SchedArr.IsZero() {
		if s := int(q.SchedArr.In(parisLoc).Sub(baseDate).Seconds()); s >= 0 {
			schedArrSec = uint32(s)
		}
	}

	isTrain := e.IsTrainRoute(q.RID)
	var tripUUID, trainNum string
	if q.TripIndex > 0 && e != nil && int(q.TripIndex-1) < len(e.Net.Trips) {
		tr := e.Net.Trips[q.TripIndex-1]
		tripUUID = extractCleanSiriID(tr.TID)
		trainNum = strings.TrimSpace(tr.SName)
	}

	var res LegLiveResult
	usedET := false
	usedSM := false

	refTime := q.SchedDep
	if refTime.IsZero() {
		refTime = nowParis
	}
	ad := e.GetActiveDelays(refTime)

	var okDep, okArr bool
	var realDepSec, realArrSec uint32
	realDepSec = schedDepSec
	realArrSec = schedArrSec

	if ad != nil {
		if cleanFrom != "" && (schedDepSec > 0 || q.TripIndex > 0) {
			if dc, ok := ad.findCallForStop(q.TripIndex, cleanRID, cleanFrom, schedDepSec, false); ok {
				okDep = true
				usedET = true
				if dc.Cancelled {
					res.Cancelled = true
				}
				res.DepDelaySec = dc.DepDelaySec
				realDepSec = applyDelayToSec(schedDepSec, dc.DepDelaySec)
				if isTrain {
					res.DeparturePlatform = cleanPlatformName(dc.DepPlatform)
					if res.DeparturePlatform == "" {
						res.DeparturePlatform = cleanPlatformName(dc.Platform)
					}
					res.IsLongTrain = dc.IsLongTrain
				}
			}
		}
		if cleanTo != "" && (schedArrSec > 0 || q.TripIndex > 0) {
			if ac, ok := ad.findCallForStop(q.TripIndex, cleanRID, cleanTo, schedArrSec, true); ok {
				okArr = true
				usedET = true
				if ac.Cancelled {
					res.Cancelled = true
				}
				res.ArrDelaySec = ac.ArrDelaySec
				realArrSec = applyDelayToSec(schedArrSec, ac.ArrDelaySec)
				if isTrain {
					res.ArrivalPlatform = cleanPlatformName(ac.ArrPlatform)
					if res.ArrivalPlatform == "" {
						res.ArrivalPlatform = cleanPlatformName(ac.Platform)
					}
					if res.IsLongTrain == nil {
						res.IsLongTrain = ac.IsLongTrain
					}
				}
			}
		}
	}

	if isTrain && cleanFrom != "" && (!okDep || res.DeparturePlatform == "") {
		calls := e.fetchStopMonitoringCalls(cleanFrom, baseDate)
		if smc, ok := findBestStopMonCall(calls, cleanRID, tripUUID, trainNum, q.Headsign, schedDepSec, false); ok {
			usedSM = true
			if smc.Cancelled {
				res.Cancelled = true
			}
			if !okDep {
				okDep = true
				res.DepDelaySec = smc.DepDelaySec
				realDepSec = applyDelayToSec(schedDepSec, smc.DepDelaySec)
			}
			if res.DeparturePlatform == "" && smc.DepPlatform != "" {
				res.DeparturePlatform = smc.DepPlatform
			}
			if res.IsLongTrain == nil {
				res.IsLongTrain = smc.IsLongTrain
			}
		}
	}
	if isTrain && cleanTo != "" && (!okArr || res.ArrivalPlatform == "") {
		calls := e.fetchStopMonitoringCalls(cleanTo, baseDate)
		if smc, ok := findBestStopMonCall(calls, cleanRID, tripUUID, trainNum, q.Headsign, schedArrSec, true); ok {
			usedSM = true
			if smc.Cancelled {
				res.Cancelled = true
			}
			if !okArr {
				okArr = true
				res.ArrDelaySec = smc.ArrDelaySec
				realArrSec = applyDelayToSec(schedArrSec, smc.ArrDelaySec)
			}
			if res.ArrivalPlatform == "" && smc.ArrPlatform != "" {
				res.ArrivalPlatform = smc.ArrPlatform
			}
			if res.IsLongTrain == nil {
				res.IsLongTrain = smc.IsLongTrain
			}
		}
	}

	if okDep && !okArr && schedArrSec > 0 {
		res.ArrDelaySec = res.DepDelaySec
		realArrSec = applyDelayToSec(schedArrSec, res.ArrDelaySec)
	} else if okArr && !okDep && schedDepSec > 0 {
		res.DepDelaySec = res.ArrDelaySec
		realDepSec = applyDelayToSec(schedDepSec, res.DepDelaySec)
	}
	if schedDepSec > 0 && schedArrSec > 0 && realArrSec < realDepSec {
		schedRide := uint32(0)
		if schedArrSec > schedDepSec {
			schedRide = schedArrSec - schedDepSec
		}
		realArrSec = realDepSec + schedRide
		res.ArrDelaySec = int(realArrSec) - int(schedArrSec)
	}

	if schedDepSec > 0 {
		t := baseDate.Add(time.Duration(realDepSec) * time.Second)
		res.RealDep = &t
	}
	if schedArrSec > 0 {
		t := baseDate.Add(time.Duration(realArrSec) * time.Second)
		res.RealArr = &t
	}

	res.Found = okDep || okArr || res.DeparturePlatform != "" || res.ArrivalPlatform != ""
	switch {
	case usedET && usedSM:
		res.Source = "mixed"
	case usedET:
		res.Source = "estimated_timetable"
	case usedSM:
		res.Source = "stop_monitoring"
	}
	return res
}
