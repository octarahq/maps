package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"maps/backend/internal/engine"
	"maps/backend/internal/osm"
	"maps/backend/internal/types"

	"github.com/gin-gonic/gin"
)

func TestHandleOSMPlatform_StLazareM14AndStNom(t *testing.T) {
	gin.SetMode(gin.TestMode)

	osm.BuildGridForTest(osm.PlatformData{
		Platforms: []osm.Platform{
			{
				Ref:   "2",
				Level: "-4",
				Mode:  "subway",
				Lat:   48.875699,
				Lon:   2.324244,
				Pts: [][2]float32{
					{48.8752704, 2.3238539},
					{48.876190, 2.324670},
				},
			},
			{
				Ref:  "2C",
				Mode: "tram",
				Lat:  48.86835,
				Lon:  2.05040,
			},
			{
				Ref:  "1;2",
				Mode: "train",
				Lat:  48.86788,
				Lon:  2.04960,
				Pts: [][2]float32{
					{48.86790, 2.04910},
					{48.86786, 2.05010},
				},
			},
		},
		StopPositions: []osm.StopPosition{
			{
				Level:       "-4",
				Mode:        "subway",
				LineRef:     "14",
				Headsigns:   []string{"Saint-Denis – Pleyel"},
				Lat:         48.8756052,
				Lon:         2.3241360,
				TanLat:      0.8606,
				TanLon:      0.7735,
				HasTangent:  true,
				HasDirected: true,
			},
			{
				Ref:         "2",
				Mode:        "train",
				LineRef:     "L",
				Headsigns:   []string{"Saint-Nom-la-Bretèche"},
				Lat:         48.86795,
				Lon:         2.05010,
				TanLat:      -0.29,
				TanLon:      -1.45,
				HasTangent:  true,
				HasDirected: true,
			},
		},
	})

	r := gin.New()
	r.GET("/osm/platform", func(c *gin.Context) {
		handleOSMPlatform(c, engine.Engine{})
	})
	r.GET("/transit/leg-live", func(c *gin.Context) {
		handleTransitLegLive(c, engine.Engine{
			Net: types.Network{
				Routes: []types.Route{
					{RID: "IDFM:C01384", SName: "14", RType: types.RouteTypeMetro},
				},
				Stops: []types.Stop{
					{SID: "IDFM:21962", Name: "Saint-Lazare", Lat: 48.875761, Long: 2.324311},
					{SID: "IDFM:21961", Name: "Madeleine", Lat: 48.87044, Long: 2.32659},
				},
			},
		})
	})

	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/osm/platform?lat=48.875761&lon=2.324311&from_lat=48.87044&from_lon=2.32659&allow_nearest=true&headsign=Saint-Denis+-+Pleyel&line=14&mode=subway", nil)
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("status %d", w1.Code)
	}
	var resp1 map[string]any
	if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
		t.Fatal(err)
	}
	if resp1["found"] != true || resp1["exit_side"] != "right" || resp1["ref"] != "2" {
		t.Fatalf("unexpected /osm/platform response: %+v", resp1)
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/transit/leg-live?rid=C01384&from_sid=21961&to_sid=21962&from_lat=48.87044&from_lon=2.32659&to_lat=48.875761&to_lon=2.324311&headsign=Saint-Denis+-+Pleyel&line=14&mode=subway", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("status %d", w2.Code)
	}
	var resp2 map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
		t.Fatal(err)
	}
	if resp2["arrival_exit_side"] != "right" || resp2["arrival_osm_ref"] != "2" {
		t.Fatalf("unexpected /transit/leg-live M14 response: %+v", resp2)
	}
}
