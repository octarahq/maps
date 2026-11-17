package engine

import (
	"maps/backend/internal/types"
	"maps/backend/internal/utils"
	"sync"
	"testing"
	"time"
)

func TestResolveSchedulePlatforms_IncludesArrivalTerminus(t *testing.T) {
	longTrue := true
	e := &Engine{
		delaysMu:   &sync.RWMutex{},
		routesByID: map[string]types.Route{"C01740": {RID: "IDFM:C01740", RType: types.RouteTypeTrain}},
		parentsByChild: map[string]string{
			"58566": "71370",
			"47954": "64382",
		},
	}

	ad := &ActiveDelays{
		engine: e,
		data: &DelaysData{
			ByTripAndStop: map[tripStopKey]RealtimeCall{
				{TripIndex: 42, CleanSID: "58566"}: {
					AimedDepSec:    16*3600 + 30*60,
					ExpectedDepSec: 16*3600 + 31*60,
					DepDelaySec:    60,
					HasDep:         true,
					Platform:       "7",
					DepPlatform:    "7",
					IsLongTrain:    &longTrue,
				},
				{TripIndex: 42, CleanSID: "47954"}: {
					AimedArrSec:    17*3600 + 5*60,
					ExpectedArrSec: 17*3600 + 7*60,
					ArrDelaySec:    120,
					HasArr:         true,
					Platform:       "2",
					ArrPlatform:    "2",
					IsLongTrain:    &longTrue,
				},
			},
		},
	}

	realDep, realArr, depDelay, arrDelay, depPlat, arrPlat, isLong, cancelled := ad.ResolveSchedulePlatforms(
		"IDFM:C01740",
		"IDFM:58566",
		"IDFM:47954",
		types.DepartureTime{
			TripIndex: 42,
			Dep:       16*3600 + 30*60,
			Arr:       17*3600 + 5*60,
		},
	)
	if cancelled {
		t.Fatalf("did not expect cancelled")
	}
	if realDep != 16*3600+31*60 || depDelay != 60 {
		t.Fatalf("unexpected dep: real=%d delay=%d", realDep, depDelay)
	}
	if realArr != 17*3600+7*60 || arrDelay != 120 {
		t.Fatalf("unexpected arr: real=%d delay=%d", realArr, arrDelay)
	}
	if depPlat != "7" || arrPlat != "2" {
		t.Fatalf("expected depPlat='7' arrPlat='2', got dep=%q arr=%q", depPlat, arrPlat)
	}
	if isLong == nil || !*isLong {
		t.Fatalf("expected isLong=true")
	}
}

func TestResolveLegLive_StopMonitoringFallbackAndCache(t *testing.T) {
	origFetch := fetchStopMonitoringRawFn
	defer func() { fetchStopMonitoringRawFn = origFetch }()

	nowParis := utils.NowInParis()
	y, m, d := nowParis.Date()
	baseDate := time.Date(y, m, d, 0, 0, 0, 0, parisLoc)

	schedDep := baseDate.Add(16*time.Hour + 30*time.Minute)
	schedArr := baseDate.Add(17*time.Hour + 5*time.Minute)

	fetchCalls := map[string]int{}
	fetchStopMonitoringRawFn = func(stopAreaID string) ([]byte, error) {
		fetchCalls[stopAreaID]++
		if stopAreaID == "71370" {
			return []byte(`{
				"Siri":{"ServiceDelivery":{"StopMonitoringDelivery":[{"MonitoredStopVisit":[{
					"MonitoredVehicleJourney":{
						"LineRef":{"value":"STIF:Line::C01740:"},
						"FramedVehicleJourneyRef":{"DatedVehicleJourneyRef":"SNCF:VJ::abc-123:LOC"},
						"VehicleJourneyName":[{"value":"134651"}],
						"MonitoredCall":{
							"AimedDepartureTime":"` + schedDep.UTC().Format(time.RFC3339) + `",
							"ExpectedDepartureTime":"` + schedDep.Add(90*time.Second).UTC().Format(time.RFC3339) + `",
							"DeparturePlatformName":{"value":"7"},
							"ArrivalPlatformName":{"value":"unknown"}
						}
					}
				}]}]}}
			}`), nil
		}
		if stopAreaID == "64382" {
			return []byte(`{
				"Siri":{"ServiceDelivery":{"StopMonitoringDelivery":[{"MonitoredStopVisit":[{
					"MonitoredVehicleJourney":{
						"LineRef":{"value":"STIF:Line::C01740:"},
						"FramedVehicleJourneyRef":{"DatedVehicleJourneyRef":"SNCF:VJ::abc-123:LOC"},
						"VehicleJourneyName":[{"value":"134651"}],
						"VehicleFeatureRef":[{"value":"longTrain"}],
						"MonitoredCall":{
							"AimedArrivalTime":"` + schedArr.UTC().Format(time.RFC3339) + `",
							"ExpectedArrivalTime":"` + schedArr.Add(2*time.Minute).UTC().Format(time.RFC3339) + `",
							"DeparturePlatformName":{"value":"1"},
							"ArrivalPlatformName":{"value":"1"}
						}
					}
				}]}]}}
			}`), nil
		}
		return nil, nil
	}

	e := Engine{
		delaysMu:     &sync.RWMutex{},
		stopMonCache: newStopMonitoringCache(),
		routesByID:   map[string]types.Route{"C01740": {RID: "IDFM:C01740", RType: types.RouteTypeTrain}},
		parentsByChild: map[string]string{
			"58566": "71370",
			"47954": "64382",
		},
		Net: types.Network{
			Trips: []types.Trip{
				{TID: "SNCF:abc-123", SName: "134651"},
			},
		},
	}

	res := e.ResolveLegLive(LegLiveQuery{
		RID:       "C01740",
		FromSID:   "58566",
		ToSID:     "47954",
		TripIndex: 1,
		SchedDep:  schedDep,
		SchedArr:  schedArr,
	})
	if !res.Found {
		t.Fatalf("expected live leg result found")
	}
	if res.DeparturePlatform != "7" || res.ArrivalPlatform != "1" {
		t.Fatalf("expected dep='7' arr='1', got dep=%q arr=%q", res.DeparturePlatform, res.ArrivalPlatform)
	}
	if res.DepDelaySec != 90 || res.ArrDelaySec != 120 {
		t.Fatalf("expected delays 90/120, got %d/%d", res.DepDelaySec, res.ArrDelaySec)
	}
	if res.IsLongTrain == nil || !*res.IsLongTrain {
		t.Fatalf("expected is_long_train=true")
	}

	_ = e.ResolveLegLive(LegLiveQuery{
		RID:       "C01740",
		FromSID:   "58566",
		ToSID:     "47954",
		TripIndex: 1,
		SchedDep:  schedDep,
		SchedArr:  schedArr,
	})
	if fetchCalls["71370"] != 1 || fetchCalls["64382"] != 1 {
		t.Fatalf("expected 1 fetch per StopArea due to cache, got %+v", fetchCalls)
	}
}
