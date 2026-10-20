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
		TargetTime:     targetTime,
		TimeType:       types.TimeTypeDeparture,
		AllowedModes:   []types.RouteType{types.RouteTypeTrain, types.RouteTypeMetro, types.RouteTypeBus, types.RouteTypeTram},
		WithCar:        false,
		WithBike:       false,
		WheelchairOnly: false,
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
		fmt.Printf("════════════════════════════════════════════════════════════\n")

		firstEdge := path[0]
		firstStop, _ := eng.SearchStopById(firstEdge.FromSID)
		distDep := utils.CoordsGetDistance(firstStop.Long, firstStop.Lat, lonFrom, latFrom)
		bikeTimeSec := uint32((distDep / 15.0) * 3600.0)
		walkTimeSec := uint32((distDep / 4.8) * 3600.0)

		lastEdge := path[len(path)-1]
		lastStop, _ := eng.SearchStopById(lastEdge.ToSID)
		distWalk := utils.CoordsGetDistance(lastStop.Long, lastStop.Lat, lonTo, latTo)
		walkArrMin := int((distWalk / 4.8) * 60.0)

		approachSec := walkTimeSec
		var bp *types.BikeParking
		if config.WithBike && distDep > 1.0 {
			var okP bool
			var distP float64
			bp, distP, okP = eng.FindNearestBikeParking(firstStop.Long, firstStop.Lat, 0.5)
			if okP {
				approachSec = bikeTimeSec
				fmt.Printf(" [BIKE ACCESS] Mairie de Feucherolles -> %s: %.2f km (~%d min @ 15 km/h)\n",
					firstStop.Name, distDep, (bikeTimeSec+59)/60)
				coverStr := "uncovered"
				if bp.Cover {
					coverStr = "covered"
				}
				payantStr := "free"
				if bp.Payant {
					payantStr = "paid"
				}
				fmt.Printf(" [BIKE PARKING] %s (%.0fm from station, %d spots, %s, %s)\n",
					bp.Name, distP*1000, bp.Capacite, coverStr, payantStr)
			} else {
				fmt.Printf(" [WALK ACCESS] Mairie de Feucherolles -> %s: %.2f km (~%d min walk)\n",
					firstStop.Name, distDep, (walkTimeSec+59)/60)
			}
		} else {
			fmt.Printf(" [WALK ACCESS] Mairie de Feucherolles -> %s: %.2f km (~%d min walk)\n",
				firstStop.Name, distDep, (walkTimeSec+59)/60)
		}

		trip := eng.EdgesToParsedTrip(path, targetTime.Add(time.Duration(approachSec)*time.Second))
		if trip != nil {
			trip.BikeParking = bp
			utils.PrintParsedTrip(trip)
		}

		fmt.Printf(" [FINAL WALK] %s -> EPITA Villejuif: %.2f km (~%d min walk)\n",
			lastStop.Name, distWalk, walkArrMin)
	}
}

