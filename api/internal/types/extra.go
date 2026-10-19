package types

import (
	"fmt"
	"strconv"
	"strings"
)

type ShapePoint struct {
	ShapeID string  `csv:"shape_id"`
	Lat     float64 `csv:"shape_pt_lat"`
	Lon     float64 `csv:"shape_pt_lon"`
	Seq     uint32  `csv:"shape_pt_sequence"`
}

type Agency struct {
	ID        string `csv:"agency_id"`
	Name      string `csv:"agency_name"`
	URL       string `csv:"agency_url"`
	Timezone  string `csv:"agency_timezone"`
	Lang      string `csv:"agency_lang"`
	Phone     string `csv:"agency_phone"`
	Email     string `csv:"agency_email"`
	FareURL   string `csv:"agency_fare_url"`
	Ticketing string `csv:"ticketing_deep_link_id"`
}

type Calendar struct {
	ServiceID string `csv:"service_id"`
	Monday    uint8  `csv:"monday"`
	Tuesday   uint8  `csv:"tuesday"`
	Wednesday uint8  `csv:"wednesday"`
	Thursday  uint8  `csv:"thursday"`
	Friday    uint8  `csv:"friday"`
	Saturday  uint8  `csv:"saturday"`
	Sunday    uint8  `csv:"sunday"`
	StartDate string `csv:"start_date"`
	EndDate   string `csv:"end_date"`
}

type CalendarDate struct {
	ServiceID     string `csv:"service_id"`
	Date          string `csv:"date"`
	ExceptionType uint8  `csv:"exception_type"`
}

type Pathway struct {
	PathwayID          string  `csv:"pathway_id"`
	FromStopID         string  `csv:"from_stop_id"`
	ToStopID           string  `csv:"to_stop_id"`
	Mode               uint8   `csv:"pathway_mode"`
	IsBidirectional    uint8   `csv:"is_bidirectional"`
	Length             float64 `csv:"length"`
	TraversalTime      uint32  `csv:"traversal_time"`
	StairCount         int32   `csv:"stair_count"`
	MaxSlope           float64 `csv:"max_slope"`
	MinWidth           float64 `csv:"min_width"`
	SignpostedAs       string  `csv:"signposted_as"`
	ReversedSignposted string  `csv:"reversed_signposted_as"`
}

type Attribution struct {
	AttributionID    string `csv:"attribution_id"`
	RouteID          string `csv:"route_id"`
	TripID           string `csv:"trip_id"`
	IsOperator       uint8  `csv:"is_operator"`
	OrganizationName string `csv:"organization_name"`
	URL              string `csv:"attribution_url"`
	Email            string `csv:"attribution_email"`
	Phone            string `csv:"attribution_phone"`
}

type BookingRule struct {
	BookingRuleID string `csv:"booking_rule_id"`
	BookingType   uint8  `csv:"booking_type"`
	Message       string `csv:"message"`
	PhoneNumber   string `csv:"phone_number"`
	InfoURL       string `csv:"info_url"`
	BookingURL    string `csv:"booking_url"`
}

type ObjectCodeExtension struct {
	ObjectType   string `csv:"object_type"`
	ObjectID     string `csv:"object_id"`
	ObjectSystem string `csv:"object_system"`
	ObjectCode   string `csv:"object_code"`
}

type TicketingDeepLink struct {
	TicketingDeepLinkID string `csv:"ticketing_deep_link_id"`
	WebURL              string `csv:"web_url"`
	AndroidIntentURI    string `csv:"android_intent_uri"`
	IOSUniversalLinkURL string `csv:"ios_universal_link_url"`
}

func ParseBikeParkingAccess(s string) bool {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "OUI":
		return true
	default:
		return false
	}
}

func ParseBikeParkingCoords(s string) (lat float64, lon float64, err error) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid coordinates format: %s", s)
	}
	lat, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, err
	}
	lon, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, err
	}
	return lat, lon, nil
}

type BikeParking struct {
	OSMID    string  `csv:"osm_id"`
	Lon      float64 `csv:"lon"`
	Lat      float64 `csv:"lat"`
	Name     string  `csv:"nom_com|nom"`
	Capacite int     `csv:"capacite"`
	Cover    bool    `csv:"couvert"`
	Payant   bool    `csv:"payant"`
}

type ExtraData struct {
	Agencies             []Agency
	Calendars            []Calendar
	CalendarDates        []CalendarDate
	Shapes               []ShapePoint
	Pathways             []Pathway
	Attributions         []Attribution
	BookingRules         []BookingRule
	ObjectCodesExtension []ObjectCodeExtension
	TicketingDeepLinks   []TicketingDeepLink
	BikeParkings         []BikeParking
}
