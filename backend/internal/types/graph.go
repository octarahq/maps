package types

type DepartureTime struct {
	Dep       uint32
	Arr       uint32
	ServiceID uint16
	TripIndex uint32
}

type Edge struct {
	FromSID      string
	ToSID        string
	Weight       int
	RID          string
	Headsign     string
	Wheelchair   Accessibility
	BikesAllowed Accessibility
	Schedules    []DepartureTime
	TripIndex    uint32
	DepTime      uint32
	ArrTime      uint32
}

type GraphType map[string][]Edge
