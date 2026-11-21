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
	Time                time.Time  `json:"time"`
	ScheduledTime       *time.Time `json:"scheduled_time,omitempty"`
	IsLongTrain         *bool      `json:"is_long_train,omitempty"`
	RID                 string     `json:"route_id"`
	RName               string     `json:"route_name"`
	RColor              string     `json:"color"`
	Picto               string     `json:"picto,omitempty"`
	Headsign            string     `json:"headsign"`
	TimeString          string     `json:"time_string"`
	ScheduledTimeString string     `json:"scheduled_time_string,omitempty"`
	Platform            string     `json:"platform,omitempty"`
	WaitMinutes         float64    `json:"wait_minutes"`
	DepSec              int        `json:"dep_sec"`
	ScheduledDepSec     int        `json:"scheduled_dep_sec,omitempty"`
	DelaySec            int        `json:"delay_sec"`
	DelayMinutes        int        `json:"delay_minutes"`
	RType               RouteType  `json:"type"`
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
	Periods          []DisruptionPeriod
	ImpactedSections []DisruptionImpactedSections
	DID              string
	Title            string
	Message          string
	SMessage         string
	Cause            DisruptionCause
	Severity         DisruptionSeverity
}

type DisruptionImpactedSections struct {
	RID  string
	From string
	To   string
}
