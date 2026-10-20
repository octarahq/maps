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
	TimeType   TimeType  `json:"time_type"` //not implemented

	AllowedModes []RouteType `json:"allowed_modes,omitempty"`

	WithCar  bool `json:"with_car"` //not implemented
	WithBike bool `json:"with_bike"`

	WheelchairOnly bool `json:"wheelchair_only"` //il faudrait demander plus quun bool (permet de proposer des arrets dans les campagnes ect ou on sait pas)
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
