package types

import (
	"time"
)

type TimeType uint8

const (
	TimeTypeDeparture TimeType = iota
	TimeTypeArrival
)

type CalculatorConfig struct {
	TargetTime time.Time `json:"target_time"`
	TimeType   TimeType  `json:"time_type"`

	AllowedModes []RouteType `json:"allowed_modes,omitempty"`

	WithCar  bool `json:"with_car"` //not implemented
	WithBike bool `json:"with_bike"`

	WheelchairOnly Accessibility `json:"wheelchair_only"`
}

type RoutesToCalculateType struct {
	Name    string
	Config  CalculatorConfig
	Factors struct {
		Walk      float64
		Wait      float64
		Transfers int
	}
}

type DepartureInfo struct {
	RID         string    `json:"route_id"`
	RName       string    `json:"route_name"`
	RColor      string    `json:"color"`
	RType       RouteType `json:"type"`
	Headsign    string    `json:"headsign"`
	DepSec      int       `json:"dep_sec"`
	Time        time.Time `json:"time"`
	TimeString  string    `json:"time_string"`
	WaitMinutes float64   `json:"wait_minutes"`
}
