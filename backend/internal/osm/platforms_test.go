package osm

import (
	"os"
	"sync"
	"testing"
)

func installStLazareAndStNomFixture() {

	tanNLat, tanNLon, _ := normalizeMetricDir(48.8756052, 48.8762320-48.8756052, 2.3246966-2.3241360)

	tanSLat, tanSLon := -tanNLat, -tanNLon

	tanLLat, tanLLon, _ := normalizeMetricDir(48.86795, -0.0002, -0.0010)

	data := PlatformData{
		Platforms: []Platform{

			{
				Ref:    "2",
				Level:  "-4",
				Mode:   "subway",
				IsEdge: false,
				Lat:    48.875699,
				Lon:    2.324244,
				Pts: [][2]float32{
					{48.8752704, 2.3238539},
					{48.876190, 2.324670},
					{48.876160, 2.324715},
					{48.8752425, 2.3239269},
					{48.8752704, 2.3238539},
				},
			},

			{
				Ref:    "1",
				Level:  "-4",
				Mode:   "subway",
				IsEdge: false,
				Lat:    48.875742,
				Lon:    2.324182,
				Pts: [][2]float32{
					{48.875340, 2.323790},
					{48.8761908, 2.3246022},
					{48.876220, 2.324555},
					{48.875370, 2.323745},
					{48.875340, 2.323790},
				},
			},

			{
				Ref:    "",
				Level:  "-4",
				Mode:   "",
				IsEdge: true,
				Lat:    48.875717,
				Lon:    2.324250,
				Pts: [][2]float32{
					{48.8752704, 2.3238539},
					{48.876190, 2.324670},
				},
			},

			{
				Ref:   "2C",
				Level: "",
				Mode:  "tram",
				Lat:   48.86835,
				Lon:   2.05040,
			},

			{
				Ref:   "1;2",
				Level: "",
				Mode:  "train",
				Lat:   48.86788,
				Lon:   2.04960,
				Pts: [][2]float32{
					{48.86790, 2.04910},
					{48.86786, 2.05010},
				},
			},

			{
				Ref:  "",
				Mode: "subway",
				Lat:  48.80005,
				Lon:  2.35008,
				Pts: [][2]float32{
					{48.80000, 2.35006},
					{48.80010, 2.35012},
				},
			},
		},
		StopPositions: []StopPosition{

			{
				Ref:         "",
				Level:       "-4",
				Mode:        "subway",
				LineRef:     "14",
				Headsigns:   []string{"Saint-Denis – Pleyel"},
				Lat:         48.8756052,
				Lon:         2.3241360,
				TanLat:      tanNLat,
				TanLon:      tanNLon,
				HasTangent:  true,
				HasDirected: true,
			},

			{
				Ref:         "",
				Level:       "-4",
				Mode:        "subway",
				LineRef:     "14",
				Headsigns:   []string{"Aéroport d’Orly"},
				Lat:         48.8756142,
				Lon:         2.3241006,
				TanLat:      tanSLat,
				TanLon:      tanSLon,
				HasTangent:  true,
				HasDirected: true,
			},

			{
				Ref:         "2",
				Mode:        "train",
				LineRef:     "L",
				Headsigns:   []string{"Saint-Nom-la-Bretèche - Forêt de Marly"},
				Lat:         48.86795,
				Lon:         2.05010,
				TanLat:      tanLLat,
				TanLon:      tanLLon,
				HasTangent:  true,
				HasDirected: true,
			},
		},
		TrackSegs: []TrackSeg{
			{
				Lat1: 48.80000,
				Lon1: 2.35000,
				Lat2: 48.80012,
				Lon2: 2.35006,
				Mode: "subway",
			},
		},
	}
	BuildGridForTest(data)
}

func TestResolvePlatform_StLazareM14CurvedApproach(t *testing.T) {
	installStLazareAndStNomFixture()

	resNorth := ResolvePlatform(PlatformQuery{
		Lat:          48.875761,
		Lon:          2.324311,
		AllowNearest: true,
		FromLat:      48.87044,
		FromLon:      2.32659,
		HasFrom:      true,
		Headsign:     "Saint-Denis - Pleyel",
		Line:         "14",
		Mode:         "subway",
	})
	if !resNorth.Found {
		t.Fatalf("expected northbound M14 Saint-Lazare platform to be found")
	}
	if resNorth.Platform.Ref != "2" {
		t.Fatalf("expected platform ref '2', got %q", resNorth.Platform.Ref)
	}
	if resNorth.ExitSide != "right" {
		t.Fatalf("expected northbound M14 Saint-Lazare exit_side='right', got %q", resNorth.ExitSide)
	}

	resNorthNoHS := ResolvePlatform(PlatformQuery{
		Lat:          48.875761,
		Lon:          2.324311,
		AllowNearest: true,
		FromLat:      48.87044,
		FromLon:      2.32659,
		HasFrom:      true,
	})
	if !resNorthNoHS.Found || resNorthNoHS.ExitSide != "right" {
		t.Fatalf("expected northbound M14 fallback exit_side='right', got found=%v side=%q", resNorthNoHS.Found, resNorthNoHS.ExitSide)
	}

	resSouth := ResolvePlatform(PlatformQuery{
		Lat:          48.875761,
		Lon:          2.324311,
		AllowNearest: true,
		FromLat:      48.8884,
		FromLon:      2.3137,
		HasFrom:      true,
		Headsign:     "Aéroport d'Orly",
		Line:         "14",
		Mode:         "subway",
	})
	if !resSouth.Found {
		t.Fatalf("expected southbound M14 Saint-Lazare platform to be found")
	}
	if resSouth.Platform.Ref != "1" {
		t.Fatalf("expected southbound platform ref '1', got %q", resSouth.Platform.Ref)
	}
	if resSouth.ExitSide != "right" {
		t.Fatalf("expected southbound M14 Saint-Lazare exit_side='right', got %q", resSouth.ExitSide)
	}
}

func TestResolvePlatform_SaintNomTrainVsTram(t *testing.T) {
	installStLazareAndStNomFixture()

	res := ResolvePlatform(PlatformQuery{
		Lat:      48.86795,
		Lon:      2.05010,
		Ref:      "2",
		FromLat:  48.8740,
		FromLon:  2.0750,
		HasFrom:  true,
		Headsign: "Saint-Nom-la-Bretèche",
		Line:     "L",
		Mode:     "train",
	})
	if !res.Found {
		t.Fatalf("expected Saint-Nom platform to be found")
	}
	if res.Platform.Mode == "tram" {
		t.Fatalf("must not match T13 tram platform %q for train line L", res.Platform.Ref)
	}
	if res.Platform.Ref != "2" {
		t.Fatalf("expected ref '2', got %q", res.Platform.Ref)
	}
	if res.ExitSide != "left" {
		t.Fatalf("expected exit_side='left' on island track 2 approaching WSW, got %q", res.ExitSide)
	}
}

func TestResolvePlatform_TrackSegFallbackWithoutStopPosition(t *testing.T) {
	installStLazareAndStNomFixture()

	res := ResolvePlatform(PlatformQuery{
		Lat:          48.80005,
		Lon:          2.35008,
		AllowNearest: true,
		FromLat:      48.79500,
		FromLon:      2.35500,
		HasFrom:      true,
		Mode:         "subway",
	})
	if !res.Found || !res.HasTrack {
		t.Fatalf("expected fallback TrackSeg match, got found=%v hasTrack=%v", res.Found, res.HasTrack)
	}
	if res.ExitSide != "right" {
		t.Fatalf("expected exit_side='right' via TrackSeg tangent, got %q", res.ExitSide)
	}
}

func TestIntegration_RealIDFPlatforms(t *testing.T) {
	if os.Getenv("OSM_INTEGRATION_CHECK") != "1" {
		t.Skip("set OSM_INTEGRATION_CHECK=1 to parse/verify real Île-de-France PBF")
	}
	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir("../../.."); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}
	loadOnce = sync.Once{}
	EnsureLoaded()

	resM14 := ResolvePlatform(PlatformQuery{
		Lat:          48.875761,
		Lon:          2.324311,
		AllowNearest: true,
		FromLat:      48.87044,
		FromLon:      2.32659,
		HasFrom:      true,
		Headsign:     "Saint-Denis - Pleyel",
		Line:         "14",
		Mode:         "subway",
	})
	t.Logf("REAL M14 St-Lazare northbound: %+v", resM14)
	if !resM14.Found || resM14.ExitSide != "right" || resM14.Platform.Ref != "2" {
		t.Fatalf("real PBF M14 St-Lazare northbound expected right & ref='2', got %+v", resM14)
	}

	expectedSides := map[string]string{"1": "right", "2": "left"}
	for _, trackRef := range []string{"1", "2"} {
		resL := ResolvePlatform(PlatformQuery{
			Lat:      48.86742,
			Lon:      2.05125,
			Ref:      trackRef,
			FromLat:  48.8670,
			FromLon:  2.0690,
			HasFrom:  true,
			Headsign: "Saint-Nom-la-Bretèche",
			Line:     "L",
			Mode:     "train",
		})
		t.Logf("REAL Saint-Nom Ligne L ref=%s: %+v", trackRef, resL)
		if !resL.Found || resL.Platform.Mode == "tram" || resL.ExitSide != expectedSides[trackRef] {
			t.Fatalf("real PBF Saint-Nom track %s unexpected: %+v", trackRef, resL)
		}
	}
}
