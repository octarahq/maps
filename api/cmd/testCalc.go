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

	path, err := engine.CalculateRouteStopToStop(from.SID, to.SID)
	if err != nil {
		panic(err)
	}

	stops := utils.IndexStopsByID(engine.Net.Stops)
	routes := utils.IndexRoutesByID(engine.Net.Routes)


	parsedTrip := utils.BuildTripFormPath(path)

	for _, step := range parsedTrip.Steps {
		rName := "Marche / Correspondance"
		if step.ViaRID != "" {
			rName = routes[step.ViaRID].SName
			if rName == "" {
				rName = routes[step.ViaRID].LName
			}
		}
		fromName := stops[step.FromSID].Name
		toName := stops[step.ToSID].Name
		if fromName == toName && step.ViaRID == "" {
			fmt.Printf("Correspondance à %s\n", fromName)
			continue
		}
		fmt.Printf("From %s To %s via %s (%d arrêts)\n", fromName, toName, rName, len(step.Stops))
	}

	fmt.Printf("Take %d sec to search trip\n", int(time.Since(startTime).Seconds()))
	startTime = time.Now()

}
