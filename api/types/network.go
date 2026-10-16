package types

import (
	"fmt"
	"strconv"
	"strings"
)

type Accessibility uint8

const (
	AccessibilityUnknown Accessibility = iota
	AccessibilityYes
	AccessibilityNo
)

type RouteType uint8

const (
	RouteTypeTram RouteType = iota
	RouteTypeMetro
	RouteTypeTrain
	RouteTypeBus
	RouteTypeFerry
	RouteTypeCableTram
	RouteTypeAerialLift
	RouteTypeFunicular
)

type TransferType uint8

const (
	TransferTypeRecommended TransferType = iota
	TransferTypeTimed
	TransferTypeNormal
	TransferTypeNotPossible
)

type ParentType uint8

const (
	ParentTypeUnknown ParentType = iota
	ParentTypeLift
	ParentTypeMetro
	ParentTypeStreetBus
	ParentTypeStreetTram
	ParentTypeRail
)

func ParseParentType(s string) ParentType {
	switch s {
	case "liftStation":
		return ParentTypeLift
	case "metroStation":
		return ParentTypeMetro
	case "onstreetBus", "busStation":
		return ParentTypeStreetBus
	case "onstreetTram", "tramStation":
		return ParentTypeStreetTram
	case "railStation":
		return ParentTypeRail
	default:
		if v, err := strconv.ParseUint(s, 10, 8); err == nil {
			return ParentType(v)
		}
		return ParentTypeUnknown
	}
}

type TrainPosition uint8

const (
	TrainPositionUnknown TrainPosition = iota
	TrainPositionFront
	TrainPositionMiddle
	TrainPositionBack
)

func ParseTrainPosition(s string) TrainPosition {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "avant", "front":
		return TrainPositionFront
	case "milieu", "middle":
		return TrainPositionMiddle
	case "arrière", "arriere", "back":
		return TrainPositionBack
	default:
		if v, err := strconv.ParseUint(s, 10, 8); err == nil {
			return TrainPosition(v)
		}
		return TrainPositionUnknown
	}
}

type GTFSTime uint32

func ParseGTFSTime(s string) (GTFSTime, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	var h, m, sec uint32
	var part int
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ':' {
			part++
			if part > 2 {
				return 0, fmt.Errorf("invalid time format: %s", s)
			}
			continue
		}
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid character in time %s: %c", s, c)
		}
		digit := uint32(c - '0')
		switch part {
		case 0:
			h = h*10 + digit
		case 1:
			m = m*10 + digit
		case 2:
			sec = sec*10 + digit
		}
	}
	if part != 2 {
		return 0, fmt.Errorf("invalid time format: %s", s)
	}
	return GTFSTime(h*3600 + m*60 + sec), nil
}

type Stop struct {
	SID           string        `csv:"stop_id"`
	Name          string        `csv:"stop_name"`
	Long          float64       `csv:"stop_lon"`
	Lat           float64       `csv:"stop_lat"`
	ZoneId        string        `csv:"zone_id"`
	ParentStation string        `csv:"parent_station"`
	WheelChair    Accessibility `csv:"wheelchair_boarding"`
}

type Route struct {
	RID      string    `csv:"route_id"`
	AgencyID string    `csv:"agency_id"`
	SName    string    `csv:"route_short_name"`
	LName    string    `csv:"route_long_name"`
	RType    RouteType `csv:"route_type"`
	Color    string    `csv:"route_color"`
	TColor   string    `csv:"route_text_color"`
}

type Trip struct {
	RID          string        `csv:"route_id"`
	TID          string        `csv:"trip_id"`
	Headsign     string        `csv:"trip_headsign"`
	SName        string        `csv:"trip_short_name"`
	DirID        uint8         `csv:"direction_id"`
	WheelChair   Accessibility `csv:"wheelchair_accessible"`
	BikesAllowed Accessibility `csv:"bikes_allowed"`
}

type StopTime struct {
	TID   string   `csv:"trip_id"`
	ATime GTFSTime `csv:"arrival_time"`
	DTime GTFSTime `csv:"departure_time"`
	SId   string   `csv:"stop_id"`
	Seq   uint16   `csv:"stop_sequence"`
}

type Transfer struct {
	FromSID  string       `csv:"from_stop_id"`
	ToSID    string       `csv:"to_stop_id"`
	TType    TransferType `csv:"transfer_type"`
	MinTTime uint16       `csv:"min_transfer_time"`
}

type ParentsStation struct {
	PID   string     `csv:"zdcid"`
	Name  string     `csv:"zdcname"`
	PType ParentType `csv:"zdctype"`
}

type PositionInTrain struct {
	FromSID     string        `csv:"from_id"`
	ToSID       string        `csv:"to_id"`
	RID         string        `csv:"line_id"`
	PositionAvg TrainPosition `csv:"position_average"`
	Pos         uint8         `csv:"position"`
	Wagons      uint8         `csv:"position_max"`
}

type Network struct {
	Stops            []Stop
	Routes           []Route
	Trips            []Trip
	StopTimes        []StopTime
	Transfers        []Transfer
	ParentsStations  []ParentsStation
	PositionsInTrain []PositionInTrain
}
