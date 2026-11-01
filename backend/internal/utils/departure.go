package utils

import (
	"cmp"
	"maps/backend/internal/types"
	"slices"
)

func FindNextDeparture(schedules []types.DepartureTime, currSec uint32, activeServices ...[]bool) (types.DepartureTime, bool) {
	idx, _ := slices.BinarySearchFunc(schedules, currSec, func(s types.DepartureTime, target uint32) int {
		return cmp.Compare(s.Dep, target)
	})

	var active []bool
	if len(activeServices) > 0 {
		active = activeServices[0]
	}

	if len(active) == 0 {
		if idx < len(schedules) {
			return schedules[idx], true
		}
		return types.DepartureTime{}, false
	}

	for idx < len(schedules) {
		s := schedules[idx]
		if int(s.ServiceID) < len(active) && active[s.ServiceID] {
			return s, true
		}
		idx++
	}
	return types.DepartureTime{}, false
}

func FindNextDepartureWithDep(schedules []types.DepartureTime, currSec uint32, activeServices ...[]bool) (uint32, uint32, bool) {
	idx, _ := slices.BinarySearchFunc(schedules, currSec, func(s types.DepartureTime, target uint32) int {
		return cmp.Compare(s.Dep, target)
	})

	var active []bool
	if len(activeServices) > 0 {
		active = activeServices[0]
	}

	if len(active) == 0 {
		if idx < len(schedules) {
			return schedules[idx].Dep, schedules[idx].Arr, true
		}
		return 0, 0, false
	}

	for idx < len(schedules) {
		s := schedules[idx]
		if int(s.ServiceID) < len(active) && active[s.ServiceID] {
			return s.Dep, s.Arr, true
		}
		idx++
	}
	return 0, 0, false
}

func FindPrevDeparture(schedules []types.DepartureTime, currSec uint32, activeServices ...[]bool) (types.DepartureTime, bool) {
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

	if bestIdx == -1 {
		return types.DepartureTime{}, false
	}

	var active []bool
	if len(activeServices) > 0 {
		active = activeServices[0]
	}

	if len(active) == 0 {
		return schedules[bestIdx], true
	}

	for bestIdx >= 0 {
		s := schedules[bestIdx]
		if int(s.ServiceID) < len(active) && active[s.ServiceID] {
			return s, true
		}
		bestIdx--
	}
	return types.DepartureTime{}, false
}

func FindPrevDepartureWithArr(schedules []types.DepartureTime, currSec uint32, activeServices ...[]bool) (uint32, uint32, bool) {
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

	if bestIdx == -1 {
		return 0, 0, false
	}

	var active []bool
	if len(activeServices) > 0 {
		active = activeServices[0]
	}

	if len(active) == 0 {
		return schedules[bestIdx].Dep, schedules[bestIdx].Arr, true
	}

	for bestIdx >= 0 {
		s := schedules[bestIdx]
		if int(s.ServiceID) < len(active) && active[s.ServiceID] {
			return s.Dep, s.Arr, true
		}
		bestIdx--
	}
	return 0, 0, false
}

func FindTripDeparture(schedules []types.DepartureTime, tripIndex uint32, currSec uint32, activeServices ...[]bool) (types.DepartureTime, bool) {
	var active []bool
	if len(activeServices) > 0 {
		active = activeServices[0]
	}
	for i := range schedules {
		s := &schedules[i]
		if s.TripIndex == tripIndex {
			if s.Dep >= currSec {
				if len(active) == 0 || (int(s.ServiceID) < len(active) && active[s.ServiceID]) {
					return *s, true
				}
			}
			return types.DepartureTime{}, false
		}
	}
	return types.DepartureTime{}, false
}

func FindTripArrival(schedules []types.DepartureTime, tripIndex uint32, currSec uint32, activeServices ...[]bool) (types.DepartureTime, bool) {
	var active []bool
	if len(activeServices) > 0 {
		active = activeServices[0]
	}
	for i := range schedules {
		s := &schedules[i]
		if s.TripIndex == tripIndex {
			if s.Arr <= currSec {
				if len(active) == 0 || (int(s.ServiceID) < len(active) && active[s.ServiceID]) {
					return *s, true
				}
			}
			return types.DepartureTime{}, false
		}
	}
	return types.DepartureTime{}, false
}
