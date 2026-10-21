package types

type DepartureTime struct {
	Dep       uint32
	Arr       uint32
	ServiceID uint16
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
}

type GraphType map[string][]Edge
