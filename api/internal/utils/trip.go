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
	fmt.Printf(" duration : %d min (%d walk) | %d transfer(s)\n", durMin, walkMin, trip.TransfersCount)
	if trip.BikeParking != nil {
		bp := trip.BikeParking
		coverStr := "uncovered"
		if bp.Cover {
			coverStr = "covered"
		}
		payantStr := "free"
		if bp.Payant {
			payantStr = "paid"
		}
		fmt.Printf(" bike parking : %s (%d spots, %s, %s)\n", bp.Name, bp.Capacite, coverStr, payantStr)
	}
	fmt.Println(strings.Repeat("─", 60))

	for i, leg := range trip.Legs {
		legDep := leg.DepartureTime.Format("15:04")
		legArr := leg.ArrivalTime.Format("15:04")
		legDur := leg.DurationSec / 60

		if leg.Type == types.LegTypeWalk {
			fmt.Printf(" [%s] walk (%d min)\n", legDep, legDur)
			fmt.Printf("        └─ from : %s (wheelchair: %t, bikes: %t)\n", leg.FromStopName, IsAccessible(leg.Wheelchair), IsAccessible(leg.BikesAllowed))
			fmt.Printf("        └─ to   : %s (wheelchair: %t, bikes: %t)\n", leg.ToStopName, IsAccessible(leg.Wheelchair), IsAccessible(leg.BikesAllowed))
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
			fmt.Printf("        ├─ from : %s (wheelchair: %t, bikes: %t)\n", leg.FromStopName, IsAccessible(leg.Wheelchair), IsAccessible(leg.BikesAllowed))

			stopCount := len(leg.IntermediateStops)
			if stopCount > 0 {
				fmt.Printf("        │  (%d stops)\n", stopCount)
			}

			fmt.Printf(" [%s] └─ to   : %s (wheelchair: %t, bikes: %t)\n", legArr, leg.ToStopName, IsAccessible(leg.Wheelchair), IsAccessible(leg.BikesAllowed))
		}

		if i < len(trip.Legs)-1 {
			fmt.Println("        │")
		}
	}

	fmt.Println(strings.Repeat("═", 60))
}
