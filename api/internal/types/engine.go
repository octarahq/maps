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
