//go:build ignore

package main

import (
	"fmt"
	"maps/api/internal/engine"
	"maps/api/internal/utils"
	"time"
)

func main() {
	startTime := time.Now()

	engine := engine.StartEngine()
	fmt.Printf("Take %d sec to load\n", int(time.Since(startTime).Seconds()))
	startTime = time.Now()

	from := engine.SearchStopsByName("Foret de marly")[0]
	fmt.Println(from.Name)
	to := engine.SearchStopsByName("Villejuif - Gustave Roussy")[0]
	fmt.Println(to.Name)

	fmt.Printf("Take %d sec to search stops\n", int(time.Since(startTime).Seconds()))
	startTime = time.Now()

	path, err := engine.CalculateRouteStopToStopWithDepartureTime(from.SID, to.SID, startTime) //"IDFM:monomodalStopPlace:58566", "IDFM:21962", startTime)
	if err != nil {
		panic(err)
	}

	trip := engine.EdgesToParsedTrip(path, startTime)

	utils.PrintParsedTrip(trip)

	fmt.Printf("Take %d ms to search trip\n", int(time.Since(startTime).Milliseconds()))
	startTime = time.Now()

}
