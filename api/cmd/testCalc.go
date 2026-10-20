//go:build ignore

package main

import (
	"fmt"
	"strings"
	"time"

	"maps/api/internal/engine"
	"maps/api/internal/types"
	"maps/api/internal/utils"
)

func main() {
	fmt.Println("\033[H\033[2J")
	startTime := time.Now()

	eng := engine.StartEngine()
	fmt.Printf("Loaded engine in %.2fs\n", time.Since(startTime).Seconds())

	lonFrom, latFrom := 1.977762, 48.879054
	lonTo, latTo := 2.357834, 48.797749

	now := time.Now()
	targetTime := time.Date(now.Year(), now.Month(), now.Day()+1, 8, 0, 0, 0, now.Location())

	config := types.CalculatorConfig{
		TargetTime:   targetTime,
		TimeType:     types.TimeTypeDeparture,
		AllowedModes: []types.RouteType{types.RouteTypeTrain, types.RouteTypeMetro, types.RouteTypeBus, types.RouteTypeTram},
		WithCar:      false,
		WithBike:     true,
	}

	searchStart := time.Now()
	routes, err := eng.CalculateRouteCordsToCordsWithDepartureTime(lonFrom, latFrom, lonTo, latTo, config)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Found %d routes in %dms\n", len(routes), time.Since(searchStart).Milliseconds())

	order := []string{"fastest", "comfort", "least_walk", "nearest_station"}
	for _, name := range order {
		path, ok := routes[name]
		if !ok || len(path) == 0 {
			continue
		}

		fmt.Printf("\n════════════════════════════════════════════════════════════\n")
		fmt.Printf(" ROUTE: %s\n", strings.ToUpper(name))

		trip := eng.EdgesToParsedTrip(path, lonFrom, latFrom, lonTo, latTo, config)
		if trip != nil {
			utils.PrintParsedTrip(trip)
		}
	}
}
