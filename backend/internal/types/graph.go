package types

type DepartureTime struct {
	Dep       uint32
	Arr       uint32
	TripIndex uint32
	ServiceID uint16
}

type Edge struct {
	Schedules    []DepartureTime
	FromSID      string
	ToSID        string
	RID          string
	Headsign     string
	Platform     string
	IsLongTrain  *bool
	Weight       int
	DepDelaySec  int
	ArrDelaySec  int
	TripIndex    uint32
	DepTime      uint32
	ArrTime      uint32
	SchedDepTime uint32
	SchedArrTime uint32
	Wheelchair   Accessibility
	BikesAllowed Accessibility
}

type GraphType map[string][]Edge
