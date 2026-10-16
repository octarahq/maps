package main

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/gob"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

var DataUrls = []struct {
	Name     string
	Url      string
	FileType string
	Path     string
}{
	{
		Name:     "gtfs",
		Url:      "https://eu.ftp.opendatasoft.com/stif/GTFS/IDFM-gtfs.zip",
		FileType: "zip",
	},
	{
		Name:     "parents_station.csv",
		Url:      "https://data.iledefrance-mobilites.fr/api/explore/v2.1/catalog/datasets/zones-de-correspondance/exports/csv",
		FileType: "csv",
	},
	{
		Name:     "position_in_train.csv",
		Url:      "https://data.iledefrance-mobilites.fr/api/explore/v2.1/catalog/datasets/positionnement-dans-la-rame/exports/csv",
		FileType: "csv",
	},
}

func main() {
	for _, d := range DataUrls {
		if d.Name == "" || d.Url == "" {
			fmt.Println("skip missing data")
			continue
		}
		fmt.Println(d.Name)
		res := fetchData(d.Url)
		var path string = fmt.Sprintf("data/raw/%s", d.Name)
		if d.Path != "" {
			path = d.Path
		}

		switch d.FileType {
		case "zip":
			fmt.Print(saveAndExtractZip(res, "data/raw"))
		default:
			fmt.Print(saveData(res, path))
		}
	}

	fmt.Println("network.bin")
	saveFile(buildNetWork("data/raw"), "data/processed/network.bin")
}

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

func parseParentType(s string) ParentType {
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

func parseTrainPosition(s string) TrainPosition {
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

func parseGTFSTime(s string) (GTFSTime, error) {
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

func buildNetWork(path string) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		stopsFile, err := readFile(fmt.Sprintf("%s/stops.txt", path))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		defer stopsFile.Close()
		stops, err := parseCSV[Stop](',', stopsFile)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		routesFile, err := readFile(fmt.Sprintf("%s/routes.txt", path))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		defer routesFile.Close()
		routes, err := parseCSV[Route](',', routesFile)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		tripsFile, err := readFile(fmt.Sprintf("%s/trips.txt", path))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		defer tripsFile.Close()
		trips, err := parseCSV[Trip](',', tripsFile)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		transfersFile, err := readFile(fmt.Sprintf("%s/transfers.txt", path))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		defer transfersFile.Close()
		transfers, err := parseCSV[Transfer](',', transfersFile)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		parentsFile, err := readFile(fmt.Sprintf("%s/parents_station.csv", path))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		defer parentsFile.Close()
		parents, err := parseCSV[ParentsStation](';', parentsFile)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		posFile, err := readFile(fmt.Sprintf("%s/position_in_train.csv", path))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		defer posFile.Close()
		positions, err := parseCSV[PositionInTrain](';', posFile)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		stopTimesFile, err := readFile(fmt.Sprintf("%s/stop_times.txt", path))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		defer stopTimesFile.Close()
		stopTimes, err := parseCSV[StopTime](',', stopTimesFile)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		net := Network{
			Stops:            stops,
			Routes:           routes,
			Trips:            trips,
			StopTimes:        stopTimes,
			Transfers:        transfers,
			ParentsStations:  parents,
			PositionsInTrain: positions,
		}

		enc := gob.NewEncoder(pw)
		if err := enc.Encode(net); err != nil {
			pw.CloseWithError(err)
			return
		}
	}()

	return pr
}

func parseCSV[T any](separator rune, data io.Reader) ([]T, error) {
	reader := csv.NewReader(data)
	reader.Comma = separator
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true
	reader.ReuseRecord = true

	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}
	if len(headers) > 0 {
		headers[0] = strings.TrimPrefix(headers[0], "\ufeff")
	}

	result := make([]T, 0)
	typeOfT := reflect.TypeFor[T]()
	valueType := typeOfT
	if valueType.Kind() == reflect.Pointer {
		valueType = valueType.Elem()
	}
	fields := make([]int, len(headers))
	for i, header := range headers {
		fields[i] = -1
		for j := 0; j < valueType.NumField(); j++ {
			field := valueType.Field(j)
			csvName := strings.Split(field.Tag.Get("csv"), ",")[0]
			if (csvName != "" && csvName != "-" && strings.EqualFold(csvName, header)) ||
				strings.EqualFold(field.Name, strings.ReplaceAll(header, "_", "")) ||
				strings.EqualFold(field.Name, header) {
				fields[i] = j
				break
			}
		}
	}

	for {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			return result, nil
		}
		if readErr != nil {
			return nil, readErr
		}
		value := reflect.New(valueType).Elem()
		for i, text := range record {
			if i >= len(fields) || fields[i] < 0 {
				continue
			}
			field := value.Field(fields[i])
			if !field.CanSet() {
				continue
			}
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}

			if field.Type() == reflect.TypeFor[ParentType]() {
				field.SetUint(uint64(parseParentType(text)))
				continue
			}
			if field.Type() == reflect.TypeFor[GTFSTime]() {
				gtfsTime, parseErr := parseGTFSTime(text)
				if parseErr != nil {
					return nil, parseErr
				}
				field.SetUint(uint64(gtfsTime))
				continue
			}
			if field.Type() == reflect.TypeFor[TrainPosition]() {
				field.SetUint(uint64(parseTrainPosition(text)))
				continue
			}

			switch field.Kind() {
			case reflect.String:
				field.SetString(text)
			case reflect.Float32, reflect.Float64:
				val, convertErr := strconv.ParseFloat(text, field.Type().Bits())
				if convertErr != nil {
					return nil, convertErr
				}
				field.SetFloat(val)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				val, convertErr := strconv.ParseInt(text, 10, field.Type().Bits())
				if convertErr != nil {
					return nil, convertErr
				}
				field.SetInt(val)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				val, convertErr := strconv.ParseUint(text, 10, field.Type().Bits())
				if convertErr != nil {
					return nil, convertErr
				}
				field.SetUint(val)
			case reflect.Bool:
				val, convertErr := strconv.ParseBool(text)
				if convertErr != nil {
					return nil, convertErr
				}
				field.SetBool(val)
			}
		}
		result = append(result, value.Interface().(T))
	}
}

func readFile(path string) (*os.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func fetchData(url string) io.Reader {
	resp, err := http.Get(url)
	if err != nil || resp == nil {
		return strings.NewReader("")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		resp.Body.Close()
		return strings.NewReader("")
	}
	return resp.Body
}

func saveData(data io.Reader, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, data)
	return err
}

func saveFile(data io.Reader, path string) error {
	return saveData(data, path)
}

func saveAndExtractZip(data io.Reader, path string) error {
	archive, err := io.ReadAll(data)
	if err != nil {
		return err
	}

	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}

	root, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for _, entry := range reader.File {
		destination := filepath.Join(root, filepath.Clean(entry.Name))
		if destination != root && !strings.HasPrefix(destination, root+string(os.PathSeparator)) {
			return fmt.Errorf("invalid ZIP entry path: %s", entry.Name)
		}

		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(destination, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			return err
		}

		input, err := entry.Open()
		if err != nil {
			return err
		}
		output, err := os.Create(destination)
		if err != nil {
			input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		inputErr := input.Close()
		outputErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		if inputErr != nil {
			return inputErr
		}
		if outputErr != nil {
			return outputErr
		}
	}

	return nil
}
