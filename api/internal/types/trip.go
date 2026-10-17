package types

type ParesedTrip struct {
	FromSID string
	ToSID   string
	Steps   []ParsedTripStep
}

type ParsedTripStep struct {
	FromSID string
	ToSID   string
	ViaRID  string
	Stops   []string
}
