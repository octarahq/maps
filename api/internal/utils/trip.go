package utils

import "maps/api/internal/types"

func BuildTripFormPath(path []types.Edge) types.ParesedTrip {
	if len(path) == 0 {
		return types.ParesedTrip{}
	}

	var parsedTrip types.ParesedTrip
	parsedTrip.FromSID = path[0].FromSID
	parsedTrip.ToSID = path[len(path)-1].ToSID

	currentStep := types.ParsedTripStep{
		FromSID: path[0].FromSID,
		ToSID:   path[0].ToSID,
		ViaRID:  path[0].RID,
		Stops:   []string{path[0].ToSID},
	}

	for i := 1; i < len(path); i++ {
		edge := path[i]
		if edge.RID == currentStep.ViaRID {
			currentStep.ToSID = edge.ToSID
			currentStep.Stops = append(currentStep.Stops, edge.ToSID)
		} else {
			parsedTrip.Steps = append(parsedTrip.Steps, currentStep)
			currentStep = types.ParsedTripStep{
				FromSID: edge.FromSID,
				ToSID:   edge.ToSID,
				ViaRID:  edge.RID,
				Stops:   []string{edge.ToSID},
			}
		}
	}
	parsedTrip.Steps = append(parsedTrip.Steps, currentStep)

	return parsedTrip
}
