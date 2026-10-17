package utils

import "maps/api/internal/types"

func BuildTripFormPath(path []types.Edge) types.ParesedTrip {
	var parsedTrip types.ParesedTrip
	parsedTrip.FromSID = path[0].FromSID
	parsedTrip.ToSID = path[len(path)-1].ToSID

	var currentStep types.ParsedTripStep
	for i := 1; i < len(path); i++ {
		prev := path[i-1]
		curr := path[i]

		if prev.RID == curr.RID {
			currentStep.FromSID = prev.FromSID
			currentStep.ToSID = prev.ToSID
			currentStep.ViaRID = curr.RID
			currentStep.Stops = append(currentStep.Stops, curr.ToSID)
		} else {
			parsedTrip.Steps = append(parsedTrip.Steps, currentStep)
			currentStep = types.ParsedTripStep{}
			currentStep.FromSID = prev.FromSID
			currentStep.ToSID = prev.ToSID
			currentStep.ViaRID = curr.RID
			currentStep.Stops = append(currentStep.Stops, curr.ToSID)
		}
	}

	return parsedTrip
}
