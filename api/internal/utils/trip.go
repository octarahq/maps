package utils

import (
	"fmt"
	"strings"

	"maps/api/internal/types"
)

func PrintParsedTrip(trip *types.ParsedTrip) {
	if trip == nil {
		fmt.Println("no trip")
		return
	}

	dep := trip.DepartureTime.Format("15:04")
	arr := trip.ArrivalTime.Format("15:04")
	durMin := trip.DurationSec / 60
	walkMin := trip.WalkingDuration / 60

	fmt.Println(strings.Repeat("═", 60))
	fmt.Printf(" trip : %s -> %s (%dh%02dm)\n", dep, arr, durMin/60, durMin%60)
	fmt.Printf(" duration : %d min (%d walking) | %d corresp.\n", durMin, walkMin, trip.TransfersCount)
	fmt.Println(strings.Repeat("─", 60))

	for i, leg := range trip.Legs {
		legDep := leg.DepartureTime.Format("15:04")
		legArr := leg.ArrivalTime.Format("15:04")
		legDur := leg.DurationSec / 60

		if leg.Type == types.LegTypeWalk {
			fmt.Printf(" [%s] walking (%d min)\n", legDep, legDur)
			fmt.Printf("        └─ from : %s\n", leg.FromStopName)
			fmt.Printf("        └─ to  : %s\n", leg.ToStopName)
		} else {
			lineName := leg.Route.ShortName
			if lineName == "" {
				lineName = leg.Route.RID
			}

			headsign := ""
			if leg.Headsign != "" {
				headsign = fmt.Sprintf(" Dir. %s", leg.Headsign)
			}

			fmt.Printf(" [%s] Line %s%s (%d min)\n", legDep, lineName, headsign, legDur)
			fmt.Printf("        ├─ From : %s\n", leg.FromStopName)

			stopCount := len(leg.IntermediateStops)
			if stopCount > 0 {
				fmt.Printf("        │  (%d stops)\n", stopCount)
			}

			fmt.Printf(" [%s] └─ To : %s\n", legArr, leg.ToStopName)
		}

		if i < len(trip.Legs)-1 {
			fmt.Println("        │")
		}
	}

	fmt.Println(strings.Repeat("═", 60))
}
