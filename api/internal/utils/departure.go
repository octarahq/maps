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

func FindPrevDeparture(schedules []types.DepartureTime, currSec uint32) (types.DepartureTime, bool) {
	if len(schedules) == 0 {
		return types.DepartureTime{}, false
	}
	low := 0
	high := len(schedules) - 1
	bestIdx := -1

	for low <= high {
		mid := low + (high-low)/2
		if schedules[mid].Arr <= currSec {
			bestIdx = mid
			low = mid + 1
		} else {
			high = mid - 1
		}
	}

	if bestIdx != -1 {
		return schedules[bestIdx], true
	}
	return types.DepartureTime{}, false
}

func FindPrevDepartureWithArr(schedules []types.DepartureTime, currSec uint32) (uint32, uint32, bool) {
	if len(schedules) == 0 {
		return 0, 0, false
	}
	low := 0
	high := len(schedules) - 1
	bestIdx := -1

	for low <= high {
		mid := low + (high-low)/2
		if schedules[mid].Arr <= currSec {
			bestIdx = mid
			low = mid + 1
		} else {
			high = mid - 1
		}
	}

	if bestIdx != -1 {
		return schedules[bestIdx].Dep, schedules[bestIdx].Arr, true
	}
	return 0, 0, false
}

