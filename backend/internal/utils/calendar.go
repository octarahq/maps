package utils

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"maps/backend/internal/types"
)

type CalendarRule struct {
	ServiceID uint16
	Days      [7]bool
	StartDate uint32
	EndDate   uint32
}

type CalendarException struct {
	ServiceID     uint16
	ExceptionType uint8
}

type CalendarIndex struct {
	mu          sync.RWMutex
	serviceToID map[string]uint16
	idToService []string
	rules       []CalendarRule
	exceptions  map[uint32][]CalendarException
}

func NewCalendarIndex(calendars []types.Calendar, calendarDates []types.CalendarDate) *CalendarIndex {
	ci := &CalendarIndex{
		serviceToID: make(map[string]uint16),
		idToService: make([]string, 1),
		rules:       make([]CalendarRule, 0, len(calendars)),
		exceptions:  make(map[uint32][]CalendarException),
	}
	ci.idToService[0] = ""

	for _, cal := range calendars {
		sID := ci.GetOrRegisterService(cal.ServiceID)
		startUint, _ := strconv.ParseUint(cal.StartDate, 10, 32)
		endUint, _ := strconv.ParseUint(cal.EndDate, 10, 32)

		var days [7]bool
		days[time.Sunday] = cal.Sunday == 1
		days[time.Monday] = cal.Monday == 1
		days[time.Tuesday] = cal.Tuesday == 1
		days[time.Wednesday] = cal.Wednesday == 1
		days[time.Thursday] = cal.Thursday == 1
		days[time.Friday] = cal.Friday == 1
		days[time.Saturday] = cal.Saturday == 1

		ci.rules = append(ci.rules, CalendarRule{
			ServiceID: sID,
			Days:      days,
			StartDate: uint32(startUint),
			EndDate:   uint32(endUint),
		})
	}

	for _, cd := range calendarDates {
		sID := ci.GetOrRegisterService(cd.ServiceID)
		dateUint, _ := strconv.ParseUint(cd.Date, 10, 32)

		dKey := uint32(dateUint)
		ci.exceptions[dKey] = append(ci.exceptions[dKey], CalendarException{
			ServiceID:     sID,
			ExceptionType: cd.ExceptionType,
		})
	}

	return ci
}

func (ci *CalendarIndex) GetOrRegisterService(serviceID string) uint16 {
	if serviceID == "" {
		return 0
	}
	ci.mu.RLock()
	id, exists := ci.serviceToID[serviceID]
	ci.mu.RUnlock()
	if exists {
		return id
	}

	ci.mu.Lock()
	defer ci.mu.Unlock()
	if id, exists = ci.serviceToID[serviceID]; exists {
		return id
	}
	newID := uint16(len(ci.idToService))
	ci.serviceToID[serviceID] = newID
	ci.idToService = append(ci.idToService, serviceID)
	return newID
}

func (ci *CalendarIndex) GetServiceString(id uint16) string {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	if int(id) < len(ci.idToService) {
		return ci.idToService[id]
	}
	return ""
}

func (ci *CalendarIndex) GetActiveServices(target time.Time) []bool {
	if ci == nil {
		return nil
	}

	ci.mu.RLock()
	totalServices := len(ci.idToService)
	ci.mu.RUnlock()

	if totalServices == 0 {
		return nil
	}

	active := make([]bool, totalServices)
	active[0] = true

	y, m, d := target.Date()
	dateUint := uint32(y*10000 + int(m)*100 + d)
	weekday := target.Weekday()

	ci.mu.RLock()
	defer ci.mu.RUnlock()

	for _, rule := range ci.rules {
		if dateUint >= rule.StartDate && dateUint <= rule.EndDate {
			if rule.Days[weekday] {
				active[rule.ServiceID] = true
			}
		}
	}

	if exList, ok := ci.exceptions[dateUint]; ok {
		for _, ex := range exList {
			switch ex.ExceptionType {
			case 1:
				active[ex.ServiceID] = true
			case 2:
				active[ex.ServiceID] = false
			}
		}
	}

	return active
}

func LoadTripServiceMapping(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	mapping := make(map[string]string, 600000)
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	if !scanner.Scan() {
		return mapping, nil
	}

	header := scanner.Text()
	headers := strings.Split(header, ",")
	sIdx, tIdx := -1, -1
	for i, h := range headers {
		clean := strings.TrimSpace(strings.TrimPrefix(h, "\ufeff"))
		if clean == "service_id" {
			sIdx = i
		} else if clean == "trip_id" {
			tIdx = i
		}
	}
	if sIdx < 0 || tIdx < 0 {
		return nil, fmt.Errorf("missing service_id or trip_id in header: %s", header)
	}

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ",")
		if len(parts) > sIdx && len(parts) > tIdx {
			mapping[parts[tIdx]] = parts[sIdx]
		}
	}
	return mapping, scanner.Err()
}
