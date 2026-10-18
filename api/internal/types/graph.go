package types

type Edge struct {
	FromSID      string
	ToSID        string
	Weight       int
	RID          string
	Headsign     string
	Wheelchair   Accessibility
	BikesAllowed Accessibility
}

type GraphType map[string][]Edge
