package engine

import (
	"maps/backend/internal/types"
	"testing"
	"time"
)

func TestGetActiveDisruptions_SectionExpansionAndParentResolution(t *testing.T) {
	now := time.Now()
	e := &Engine{
		parentsByChild: map[string]string{
			"sp_nation":  "sa_nation",
			"sp_vinc":    "sa_vinc",
			"sp_font":    "sa_font",
			"sp_neuilly": "sa_neuilly",
			"sp_valdf":   "sa_valdf",
		},
		childrenByParent: map[string][]string{
			"sa_nation":  {"sp_nation"},
			"sa_vinc":    {"sp_vinc"},
			"sa_font":    {"sp_font"},
			"sa_neuilly": {"sp_neuilly"},
			"sa_valdf":   {"sp_valdf"},
		},
		Graph: types.GraphType{
			"sp_nation": {
				{RID: "IDFM:C01742", FromSID: "sp_nation", ToSID: "sp_vinc"},
			},
			"sp_vinc": {
				{RID: "IDFM:C01742", FromSID: "sp_vinc", ToSID: "sp_nation"},
				{RID: "IDFM:C01742", FromSID: "sp_vinc", ToSID: "sp_font"},
				{RID: "IDFM:C01742", FromSID: "sp_vinc", ToSID: "sp_valdf"},
			},
			"sp_font": {
				{RID: "IDFM:C01742", FromSID: "sp_font", ToSID: "sp_vinc"},
			},
			"sp_valdf": {
				{RID: "IDFM:C01742", FromSID: "sp_valdf", ToSID: "sp_vinc"},
				{RID: "IDFM:C01742", FromSID: "sp_valdf", ToSID: "sp_neuilly"},
			},
			"sp_neuilly": {
				{RID: "IDFM:C01742", FromSID: "sp_neuilly", ToSID: "sp_valdf"},
			},
		},
		Disruptions: []types.Disruption{
			{
				DID:      "test-rer-a",
				Severity: types.DisruptionSeverityBloqued,
				Periods: []types.DisruptionPeriod{
					{Begin: now.Add(-time.Hour), End: now.Add(time.Hour)},
				},
				ImpactedSections: []types.DisruptionImpactedSections{
					{RID: "C01742", From: "sa_vinc", To: "sa_neuilly"},
					{RID: "C01742", From: "sa_vinc", To: "sa_font"},
				},
			},
		},
	}

	ad := e.GetActiveDisruptions(now)

	if ad.IsEdgeBlocked(&types.Edge{RID: "IDFM:C01742", FromSID: "sp_nation", ToSID: "sp_vinc"}) {
		t.Fatalf("expected Nation -> Vincennes to remain open")
	}
	if ad.IsEdgeBlocked(&types.Edge{RID: "IDFM:C01742", FromSID: "sp_vinc", ToSID: "sp_nation"}) {
		t.Fatalf("expected Vincennes -> Nation to remain open")
	}

	if !ad.IsEdgeBlocked(&types.Edge{RID: "IDFM:C01742", FromSID: "sp_vinc", ToSID: "sp_valdf"}) {
		t.Fatalf("expected Vincennes -> Val de Fontenay to be blocked")
	}
	if !ad.IsEdgeBlocked(&types.Edge{RID: "IDFM:C01742", FromSID: "sp_valdf", ToSID: "sp_neuilly"}) {
		t.Fatalf("expected Val de Fontenay -> Neuilly to be blocked")
	}
	if !ad.IsStopBlockedOnLine("sp_valdf", "IDFM:C01742") {
		t.Fatalf("expected intermediate stop Val de Fontenay to be blocked on C01742")
	}

	if !ad.IsEdgeBlocked(&types.Edge{RID: "IDFM:C01742", FromSID: "sp_vinc", ToSID: "sp_font"}) {
		t.Fatalf("expected Vincennes -> Fontenay-sous-Bois to be blocked")
	}
	if !ad.IsEdgeBlocked(&types.Edge{RID: "IDFM:C01742", FromSID: "sp_font", ToSID: "sp_vinc"}) {
		t.Fatalf("expected Fontenay-sous-Bois -> Vincennes to be blocked")
	}
}

func TestParseDisruptions_BulkLinesImpactedObjects(t *testing.T) {
	now := time.Now()
	begin := now.Add(-time.Hour).Format("20060102T150405")
	end := now.Add(time.Hour).Format("20060102T150405")

	payload := struct {
		Disruptions []DepartureRes `json:"disruptions"`
		Lines       []BulkLineRes  `json:"lines"`
	}{
		Disruptions: []DepartureRes{
			{
				Id:       "d-stop",
				Severity: "BLOQUANTE",
				Cause:    "TRAVAUX",
				ApplicationPeriods: []struct {
					Begin string `json:"begin"`
					End   string `json:"end"`
				}{{Begin: begin, End: end}},
			},
			{
				Id:       "d-line",
				Severity: "BLOQUANTE",
				Cause:    "TRAVAUX",
				ApplicationPeriods: []struct {
					Begin string `json:"begin"`
					End   string `json:"end"`
				}{{Begin: begin, End: end}},
			},
		},
		Lines: []BulkLineRes{
			{
				ID: "line:IDFM:C01282",
				ImpactedObjects: []struct {
					Type          string   `json:"type"`
					ID            string   `json:"id"`
					DisruptionIDs []string `json:"disruptionIds"`
				}{
					{Type: "stop_point", ID: "stop_point:IDFM:29359", DisruptionIDs: []string{"d-stop"}},
					{Type: "line", ID: "line:IDFM:C01282", DisruptionIDs: []string{"d-stop"}},
				},
			},
			{
				ID: "line:IDFM:C01383",
				ImpactedObjects: []struct {
					Type          string   `json:"type"`
					ID            string   `json:"id"`
					DisruptionIDs []string `json:"disruptionIds"`
				}{
					{Type: "line", ID: "line:IDFM:C01383", DisruptionIDs: []string{"d-line"}},
				},
			},
		},
	}

	parsed := parseDisruptions(payload)
	e := &Engine{Disruptions: parsed}
	ad := e.GetActiveDisruptions(now)

	if !ad.IsStopBlockedOnLine("29359", "IDFM:C01282") {
		t.Fatalf("expected stop 29359 to be blocked on C01282")
	}
	if ad.IsEdgeBlocked(&types.Edge{RID: "IDFM:C01282", FromSID: "10001", ToSID: "10002"}) {
		t.Fatalf("expected other stops on C01282 to remain open")
	}

	if !ad.IsEdgeBlocked(&types.Edge{RID: "IDFM:C01383", FromSID: "anyA", ToSID: "anyB"}) {
		t.Fatalf("expected entire line C01383 to be blocked")
	}
}
