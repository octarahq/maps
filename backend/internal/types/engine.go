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
	RID                 string     `json:"route_id"`
	RName               string     `json:"route_name"`
	RColor              string     `json:"color"`
	RType               RouteType  `json:"type"`
	Picto               string     `json:"picto,omitempty"`
	Headsign            string     `json:"headsign"`
	DepSec              int        `json:"dep_sec"`
	Time                time.Time  `json:"time"`
	TimeString          string     `json:"time_string"`
	WaitMinutes         float64    `json:"wait_minutes"`
	ScheduledDepSec     int        `json:"scheduled_dep_sec,omitempty"`
	ScheduledTime       *time.Time `json:"scheduled_time,omitempty"`
	ScheduledTimeString string     `json:"scheduled_time_string,omitempty"`
	DelaySec            int        `json:"delay_sec"`
	DelayMinutes        int        `json:"delay_minutes"`
	IsDelayed           bool       `json:"is_delayed"`
}

type DisruptionCause uint8

const (
	DisruptionCauseUnknow DisruptionCause = iota
	DisruptionCauseTravaux
	DisruptionCauseDisturbance
	DisruptionCauseInfo
)

type DisruptionSeverity uint8

const (
	DisruptionSeverityUnknow DisruptionSeverity = iota
	DisruptionSeverityBloqued
	DisruptionSeverityPerturbate
	DisruptionSeverityInfo
)

type DisruptionPeriod struct {
	Begin time.Time
	End   time.Time
}

type Disruption struct {
	DID              string
	Periods          []DisruptionPeriod
	Cause            DisruptionCause
	Severity         DisruptionSeverity
	Title            string
	Message          string
	SMessage         string
	ImpactedSections []DisruptionImpactedSections
}

type DisruptionImpactedSections struct {
	RID  string
	From string
	To   string
}
