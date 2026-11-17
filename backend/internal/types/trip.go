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
	DepartureTime          time.Time           `json:"departure_time"`
	ArrivalTime            time.Time           `json:"arrival_time"`
	ScheduledDepartureTime *time.Time          `json:"scheduled_departure_time,omitempty"`
	ScheduledArrivalTime   *time.Time          `json:"scheduled_arrival_time,omitempty"`
	Route                  *RouteSummary       `json:"route,omitempty"`
	IsLongTrain            *bool               `json:"is_long_train,omitempty"`
	IntermediateStops      []IntermediateStop  `json:"intermediate_stops,omitempty"`
	Coordinates            [][2]float64        `json:"coordinates,omitempty"`
	FromSID                string              `json:"from_sid"`
	ToSID                  string              `json:"to_sid"`
	FromStopName           string              `json:"from_stop_name"`
	ToStopName             string              `json:"to_stop_name"`
	Headsign               string              `json:"headsign,omitempty"`
	DeparturePlatform      string              `json:"departure_platform,omitempty"`
	ArrivalPlatform        string              `json:"arrival_platform,omitempty"`
	FromLat                float64             `json:"from_lat,omitempty"`
	FromLon                float64             `json:"from_lon,omitempty"`
	ToLat                  float64             `json:"to_lat,omitempty"`
	ToLon                  float64             `json:"to_lon,omitempty"`
	DistanceKm             float64             `json:"distance_km,omitempty"`
	DepartureDelaySec      int                 `json:"departure_delay_sec,omitempty"`
	ArrivalDelaySec        int                 `json:"arrival_delay_sec,omitempty"`
	DurationSec            int                 `json:"duration_sec"`
	TripIndex              uint32              `json:"trip_index,omitempty"`
	PositionInTrain        TripPositionInTrain `json:"position_in_train"`
	Type                   LegType             `json:"type"`
	Wheelchair             Accessibility       `json:"wheelchair"`
	BikesAllowed           Accessibility       `json:"bikes_allowed"`
	IsDelayed              bool                `json:"is_delayed,omitempty"`
}

type LegType uint8

const (
	LegTypeWalk LegType = iota
	LegTypeTransit
	LegTypeBike
	LegTypeTransfer
)

type RouteSummary struct {
	RID             string    `json:"rid"`
	ShortName       string    `json:"short_name"`
	LongName        string    `json:"long_name"`
	RouteType       RouteType `json:"route_type"`
	Color           string    `json:"color"`
	TextColor       string    `json:"text_color"`
	Picto           string    `json:"picto,omitempty"`
	VehicleImage    string    `json:"vehicle_image,omitempty"`
	VehicleCarriage string    `json:"vehicle_carriage,omitempty"`
}

type IntermediateStop struct {
	ArrivalTime            time.Time  `json:"arrival_time"`
	DepartureTime          time.Time  `json:"departure_time"`
	ScheduledArrivalTime   *time.Time `json:"scheduled_arrival_time,omitempty"`
	ScheduledDepartureTime *time.Time `json:"scheduled_departure_time,omitempty"`
	SID                    string     `json:"sid"`
	Name                   string     `json:"name"`
	Lat                    float64    `json:"lat,omitempty"`
	Lon                    float64    `json:"lon,omitempty"`
	DelaySec               int        `json:"delay_sec,omitempty"`
	IsDelayed              bool       `json:"is_delayed,omitempty"`
	IsBlocked              bool       `json:"is_blocked,omitempty"`
}

type TripPositionInTrain struct {
	NbWagons uint8         `json:"nb_wagons"`
	Wagon    uint8         `json:"wagon"`
	PosAvg   TrainPosition `json:"pos_avg"`
}
