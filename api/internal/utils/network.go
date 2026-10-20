package utils

import (
	"encoding/gob"
	"fmt"
	"os"

	"maps/api/internal/types"
)

func LoadNetwork(path ...string) (*types.Network, error) {
	filePath := "data/processed/network.bin"
	if len(path) > 0 && path[0] != "" {
		filePath = path[0]
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var net types.Network
	dec := gob.NewDecoder(file)
	if err := dec.Decode(&net); err != nil {
		return nil, err
	}

	return &net, nil
}

func LoadExtra(path ...string) (*types.ExtraData, error) {
	filePath := "data/processed/extra.bin"
	if len(path) > 0 && path[0] != "" {
		filePath = path[0]
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var extra types.ExtraData
	dec := gob.NewDecoder(file)
	if err := dec.Decode(&extra); err != nil {
		return nil, err
	}

	return &extra, nil
}

func IndexStopsByID(stops []types.Stop) map[string]types.Stop {
	m := make(map[string]types.Stop, len(stops))
	for _, s := range stops {
		m[s.SID] = s
	}
	return m
}
func IndexRoutesByID(routes []types.Route) map[string]types.Route {
	m := make(map[string]types.Route, len(routes))
	for _, r := range routes {
		m[r.RID] = r
	}
	return m
}

func IndexTripsByID(trips []types.Trip) map[string]types.Trip {
	m := make(map[string]types.Trip, len(trips))
	for _, t := range trips {
		m[t.TID] = t
	}
	return m
}

func IndexStopTimesByTripID(stopTimes []types.StopTime) map[string][]types.StopTime {
	m := make(map[string][]types.StopTime)
	for _, st := range stopTimes {
		m[st.TID] = append(m[st.TID], st)
	}
	return m
}

func IndexTransfersByFromStop(transfers []types.Transfer) map[string][]types.Transfer {
	m := make(map[string][]types.Transfer)
	for _, tr := range transfers {
		m[tr.FromSID] = append(m[tr.FromSID], tr)
	}
	return m
}

func IndexPositionInTrainsByTripKey(positionInTrain []types.PositionInTrain) map[string]types.PositionInTrain {
	m := make(map[string]types.PositionInTrain)
	for _, pit := range positionInTrain {
		m[fmt.Sprintf("%s-%s;%s", pit.FromSID, pit.ToSID, pit.RID)] = pit
	}
	return m
}
