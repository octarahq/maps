package types

import "time"

type ParsedTrip struct {
	FromSID         string        `json:"from_sid"`
	ToSID           string        `json:"to_sid"`
	DepartureTime   time.Time     `json:"departure_time"`
	ArrivalTime     time.Time     `json:"arrival_time"`
	DurationSec     int           `json:"duration_sec"`
	WalkingDuration int           `json:"walking_duration_sec"`
	TransfersCount  int           `json:"transfers_count"`
	IsWheelchair    Accessibility `json:"is_wheelchair"`
	IsBikesAllowed  Accessibility `json:"is_bikes_allowed"`
	Legs            []TripLeg     `json:"legs"`
}

type TripLeg struct {
	Type          LegType   `json:"type"`
	FromSID       string    `json:"from_sid"`
	ToSID         string    `json:"to_sid"`
	FromStopName  string    `json:"from_stop_name"`
	ToStopName    string    `json:"to_stop_name"`
	DepartureTime time.Time `json:"departure_time"`
	ArrivalTime   time.Time `json:"arrival_time"`
	DurationSec   int       `json:"duration_sec"`

	Route             *RouteSummary      `json:"route,omitempty"`
	Headsign          string             `json:"headsign,omitempty"`
	IntermediateStops []IntermediateStop `json:"intermediate_stops,omitempty"`

	Wheelchair   Accessibility `json:"wheelchair"`
	BikesAllowed Accessibility `json:"bikes_allowed"`
}

type LegType uint8

const (
	LegTypeWalk LegType = iota
	LegTypeTransit
)

type RouteSummary struct {
	RID       string    `json:"rid"`
	ShortName string    `json:"short_name"`
	LongName  string    `json:"long_name"`
	RouteType RouteType `json:"route_type"`
	Color     string    `json:"color"`
	TextColor string    `json:"text_color"`
}

type IntermediateStop struct {
	SID           string    `json:"sid"`
	Name          string    `json:"name"`
	ArrivalTime   time.Time `json:"arrival_time"`
	DepartureTime time.Time `json:"departure_time"`
}
