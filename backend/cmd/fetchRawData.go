package cmd

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

	"maps/backend/internal/types"
	"maps/backend/internal/utils"
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
	{
		Name:     "bikes_parking.csv",
		Url:      "https://data.iledefrance-mobilites.fr/api/explore/v2.1/catalog/datasets/stationnement-velo-en-ile-de-france/exports/csv",
		FileType: "csv",
	},
	{
		Name:     "lines.csv",
		Url:      "https://data.iledefrance-mobilites.fr/api/explore/v2.1/catalog/datasets/referentiel-des-lignes/exports/csv?limit=10000",
		FileType: "csv",
	},
}

func RunFetchData() {
	main()
}

func main() {
	skipFetch := false
	if len(os.Args) > 1 && (os.Args[1] == "--skip-fetch") {
		skipFetch = true
		fmt.Println("skiping download")
	}

	if !skipFetch {
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
	}

	fmt.Println("network.bin")
	saveFile(buildNetWork("data/raw"), "data/processed/network.bin")
	fmt.Println("extra.bin")
	saveFile(buildExtraData("data/raw"), "data/processed/extra.bin")
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
		stops, err := parseCSV[types.Stop](',', stopsFile)
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
		routes, err := parseCSV[types.Route](',', routesFile)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		linesPath := fmt.Sprintf("%s/lines.csv", path)
		if _, statErr := os.Stat(linesPath); os.IsNotExist(statErr) {
			linesPath = fmt.Sprintf("%s/referentiel-des-lignes.csv", path)
		}
		if linesFile, err := readFile(linesPath); err == nil {
			if lines, err := parseCSV[types.LineReferentiel](';', linesFile); err == nil {
				utils.EnrichRoutesWithPictos(routes, lines)
			}
			linesFile.Close()
		}

		tripsFile, err := readFile(fmt.Sprintf("%s/trips.txt", path))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		defer tripsFile.Close()
		trips, err := parseCSV[types.Trip](',', tripsFile)
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
		transfers, err := parseCSV[types.Transfer](',', transfersFile)
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
		parents, err := parseCSV[types.ParentsStation](';', parentsFile)
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
		positions, err := parseCSV[types.PositionInTrain](';', posFile)
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
		stopTimes, err := parseCSV[types.StopTime](',', stopTimesFile)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		net := types.Network{
			Stops:            stops,
			Routes:           routes,
			Trips:            trips,
			StopTimes:        stopTimes,
			Transfers:        transfers,
			ParentsStations:  parents,
			PositionsInTrain: positions,
		}

		utils.EnrichParentsStations(&net)

		enc := gob.NewEncoder(pw)
		if err := enc.Encode(net); err != nil {
			pw.CloseWithError(err)
			return
		}
	}()

	return pr
}

func buildExtraData(path string) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		var extra types.ExtraData

		if f, err := readFile(fmt.Sprintf("%s/agency.txt", path)); err == nil {
			if data, err := parseCSV[types.Agency](',', f); err == nil {
				extra.Agencies = data
			}
			f.Close()
		}

		if f, err := readFile(fmt.Sprintf("%s/calendar.txt", path)); err == nil {
			if data, err := parseCSV[types.Calendar](',', f); err == nil {
				extra.Calendars = data
			}
			f.Close()
		}

		if f, err := readFile(fmt.Sprintf("%s/calendar_dates.txt", path)); err == nil {
			if data, err := parseCSV[types.CalendarDate](',', f); err == nil {
				extra.CalendarDates = data
			}
			f.Close()
		}

		if f, err := readFile(fmt.Sprintf("%s/shapes.txt", path)); err == nil {
			if data, err := parseCSV[types.ShapePoint](',', f); err == nil {
				extra.Shapes = data
			}
			f.Close()
		}

		if f, err := readFile(fmt.Sprintf("%s/pathways.txt", path)); err == nil {
			if data, err := parseCSV[types.Pathway](',', f); err == nil {
				extra.Pathways = data
			}
			f.Close()
		}

		if f, err := readFile(fmt.Sprintf("%s/attributions.txt", path)); err == nil {
			if data, err := parseCSV[types.Attribution](',', f); err == nil {
				extra.Attributions = data
			}
			f.Close()
		}

		if f, err := readFile(fmt.Sprintf("%s/booking_rules.txt", path)); err == nil {
			if data, err := parseCSV[types.BookingRule](',', f); err == nil {
				extra.BookingRules = data
			}
			f.Close()
		}

		if f, err := readFile(fmt.Sprintf("%s/object_codes_extension.txt", path)); err == nil {
			if data, err := parseCSV[types.ObjectCodeExtension](',', f); err == nil {
				extra.ObjectCodesExtension = data
			}
			f.Close()
		}

		if f, err := readFile(fmt.Sprintf("%s/ticketing_deep_links.txt", path)); err == nil {
			if data, err := parseCSV[types.TicketingDeepLink](',', f); err == nil {
				extra.TicketingDeepLinks = data
			}
			f.Close()
		}

		if f, err := readFile(fmt.Sprintf("%s/bikes_parking.csv", path)); err == nil {
			if data, err := parseCSV[types.BikeParking](';', f); err == nil {
				extra.BikeParkings = data
			}
			f.Close()
		}

		enc := gob.NewEncoder(pw)
		if err := enc.Encode(extra); err != nil {
			pw.CloseWithError(err)
			return
		}
	}()

	return pr
}

// func buildExtra(path string) io.Reader {
// 	return buildExtraData(path)
// }

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
			csvTag := strings.Split(field.Tag.Get("csv"), ",")[0]
			matched := false
			if csvTag != "" && csvTag != "-" {
				for _, candidate := range strings.Split(csvTag, "|") {
					if strings.EqualFold(strings.TrimSpace(candidate), header) {
						fields[i] = j
						matched = true
						break
					}
				}
			}
			if matched {
				break
			}
			if strings.EqualFold(field.Name, strings.ReplaceAll(header, "_", "")) ||
				strings.EqualFold(field.Name, header) {
				fields[i] = j
				break
			}
		}
	}

	geoPointCol := -1
	for i, header := range headers {
		clean := strings.ToLower(strings.TrimSpace(header))
		if clean == "geo_point_2d" || clean == "geo_point" || clean == "geopoint" {
			geoPointCol = i
			break
		}
	}
	latFieldIdx := -1
	lonFieldIdx := -1
	for j := 0; j < valueType.NumField(); j++ {
		name := strings.ToLower(valueType.Field(j).Name)
		if name == "lat" || name == "latitude" {
			latFieldIdx = j
		} else if name == "lon" || name == "long" || name == "longitude" {
			lonFieldIdx = j
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

		if geoPointCol >= 0 && geoPointCol < len(record) && latFieldIdx >= 0 && lonFieldIdx >= 0 {
			if lat, lon, err := types.ParseBikeParkingCoords(record[geoPointCol]); err == nil {
				value.Field(latFieldIdx).SetFloat(lat)
				value.Field(lonFieldIdx).SetFloat(lon)
			}
		}

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

			if field.Type() == reflect.TypeFor[types.ParentType]() {
				field.SetUint(uint64(types.ParseParentType(text)))
				continue
			}
			if field.Type() == reflect.TypeFor[types.GTFSTime]() {
				gtfsTime, parseErr := types.ParseGTFSTime(text)
				if parseErr != nil {
					return nil, parseErr
				}
				field.SetUint(uint64(gtfsTime))
				continue
			}
			if field.Type() == reflect.TypeFor[types.TrainPosition]() {
				field.SetUint(uint64(types.ParseTrainPosition(text)))
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
					field.SetBool(types.ParseBikeParkingAccess(text))
					continue
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
