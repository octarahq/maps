package utils

import (
	"cmp"
	"maps/api/internal/types"
	"slices"
)

func FindNextDeparture(schedules []types.DepartureTime, currSec uint32) (types.DepartureTime, bool) {
	idx, _ := slices.BinarySearchFunc(schedules, currSec, func(s types.DepartureTime, target uint32) int {
		return cmp.Compare(s.Dep, target)
	})
	if idx < len(schedules) {
		return schedules[idx], true
	}
	return types.DepartureTime{}, false
}

func FindNextDepartureWithDep(schedules []types.DepartureTime, currSec uint32) (uint32, uint32, bool) {
	idx, _ := slices.BinarySearchFunc(schedules, currSec, func(s types.DepartureTime, target uint32) int {
		return cmp.Compare(s.Dep, target)
	})
	if idx < len(schedules) {
		return schedules[idx].Dep, schedules[idx].Arr, true
	}
	return 0, 0, false
}
