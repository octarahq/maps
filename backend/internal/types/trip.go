package types

import "time"

type ParsedTrip struct {
	FromSID          string        `json:"from_sid"`
	ToSID            string        `json:"to_sid"`
	DepartureTime    time.Time     `json:"departure_time"`
	ArrivalTime      time.Time     `json:"arrival_time"`
	DurationSec      int           `json:"duration_sec"`
	WalkingDuration  int           `json:"walking_duration_sec"`
	BikeDuration     int           `json:"bike_duration_sec,omitempty"`
	TransfersCount   int           `json:"transfers_count"`
	IsWheelchair     Accessibility `json:"is_wheelchair"`
	IsBikesAllowed   Accessibility `json:"is_bikes_allowed"`
	BikeParking      *BikeParking  `json:"bike_parking,omitempty"`
	BikeParkingDistM float64       `json:"bike_parking_dist_m,omitempty"`
	Legs             []TripLeg     `json:"legs"`
}

type TripLeg struct {
	Type          LegType   `json:"type"`
	TripIndex     uint32    `json:"trip_index,omitempty"`
	FromSID       string    `json:"from_sid"`
	ToSID         string    `json:"to_sid"`
	FromStopName  string    `json:"from_stop_name"`
	ToStopName    string    `json:"to_stop_name"`
	FromLat       float64   `json:"from_lat,omitempty"`
	FromLon       float64   `json:"from_lon,omitempty"`
	ToLat         float64   `json:"to_lat,omitempty"`
	ToLon         float64   `json:"to_lon,omitempty"`
	DepartureTime time.Time `json:"departure_time"`
	ArrivalTime   time.Time `json:"arrival_time"`
	DurationSec   int       `json:"duration_sec"`
	DistanceKm    float64   `json:"distance_km,omitempty"`

	Route             *RouteSummary      `json:"route,omitempty"`
	Headsign          string             `json:"headsign,omitempty"`
	IntermediateStops []IntermediateStop `json:"intermediate_stops,omitempty"`

	PositionInTrain TripPositionInTrain `json:"position_in_train"`

	Wheelchair   Accessibility `json:"wheelchair"`
	BikesAllowed Accessibility `json:"bikes_allowed"`

	Coordinates [][2]float64 `json:"coordinates,omitempty"`
}

type LegType uint8

const (
	LegTypeWalk LegType = iota
	LegTypeTransit
	LegTypeBike
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
	IsBlocked     bool      `json:"is_blocked,omitempty"`
	Lat           float64   `json:"lat,omitempty"`
	Lon           float64   `json:"lon,omitempty"`
}

type TripPositionInTrain struct {
	NbWagons uint8         `json:"nb_wagons"`
	Wagon    uint8         `json:"wagon"`
	PosAvg   TrainPosition `json:"pos_avg"`
}
