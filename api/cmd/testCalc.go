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
	depTarget := time.Date(now.Year(), now.Month(), now.Day()+1, 8, 0, 0, 0, now.Location())

	configDep := types.CalculatorConfig{
		TargetTime:   depTarget,
		TimeType:     types.TimeTypeDeparture,
		AllowedModes: []types.RouteType{types.RouteTypeTrain, types.RouteTypeMetro, types.RouteTypeBus, types.RouteTypeTram},
		WithCar:      false,
		WithBike:     true,
	}

	fmt.Println("\n============================================================")
	fmt.Printf(" test 1 departure time at %s\n", depTarget.Format("15:04"))
	fmt.Println("============================================================")

	searchStart := time.Now()
	routesDep, err := eng.CalculateRouteCordsToCords(lonFrom, latFrom, lonTo, latTo, configDep)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Found %d routes in %dms\n", len(routesDep), time.Since(searchStart).Milliseconds())

	order := []string{"fastest", "comfort", "least_walk", "nearest_station"}
	for _, name := range order {
		path, ok := routesDep[name]
		if !ok || len(path) == 0 {
			continue
		}

		fmt.Printf("\n════════════════════════════════════════════════════════════\n")
		fmt.Printf(" ROUTE (DEPARTURE): %s\n", strings.ToUpper(name))

		trip := eng.EdgesToParsedTrip(path, lonFrom, latFrom, lonTo, latTo, configDep)
		if trip != nil {
			utils.PrintParsedTrip(trip)
		}
	}

	arrTarget := time.Date(now.Year(), now.Month(), now.Day()+1, 9, 30, 0, 0, now.Location())
	configArr := types.CalculatorConfig{
		TargetTime:   arrTarget,
		TimeType:     types.TimeTypeArrival,
		AllowedModes: []types.RouteType{types.RouteTypeTrain, types.RouteTypeMetro, types.RouteTypeBus, types.RouteTypeTram},
		WithCar:      false,
		WithBike:     true,
	}

	fmt.Println("\n============================================================")
	fmt.Printf(" test 2 arrival time at %s\n", arrTarget.Format("15:04"))
	fmt.Println("============================================================")

	searchStartArr := time.Now()
	routesArr, err := eng.CalculateRouteCordsToCords(lonFrom, latFrom, lonTo, latTo, configArr)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Found %d routes in %dms\n", len(routesArr), time.Since(searchStartArr).Milliseconds())

	for _, name := range order {
		path, ok := routesArr[name]
		if !ok || len(path) == 0 {
			continue
		}

		fmt.Printf("\n════════════════════════════════════════════════════════════\n")
		fmt.Printf(" ROUTE (ARRIVAL): %s\n", strings.ToUpper(name))

		trip := eng.EdgesToParsedTrip(path, lonFrom, latFrom, lonTo, latTo, configArr)
		if trip != nil {
			utils.PrintParsedTrip(trip)
		}
	}
}
