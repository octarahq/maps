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
	bikeMin := trip.BikeDuration / 60

	fmt.Println(strings.Repeat("═", 60))
	fmt.Printf(" trip : %s -> %s (%dh%02dm)\n", dep, arr, durMin/60, durMin%60)
	durDetail := fmt.Sprintf("%d walk", walkMin)
	if bikeMin > 0 {
		durDetail = fmt.Sprintf("%d walk, %d bike", walkMin, bikeMin)
	}
	fmt.Printf(" duration : %d min (%s) | %d transfer(s)\n", durMin, durDetail, trip.TransfersCount)
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
		distStr := ""
		if trip.BikeParkingDistM > 0 {
			distStr = fmt.Sprintf("%.0fm from station, ", trip.BikeParkingDistM)
		}
		fmt.Printf(" bike parking : %s (%s%d spots, %s, %s)\n", bp.Name, distStr, bp.Capacite, coverStr, payantStr)
	}
	fmt.Println(strings.Repeat("─", 60))

	for i, leg := range trip.Legs {
		legDep := leg.DepartureTime.Format("15:04")
		legArr := leg.ArrivalTime.Format("15:04")
		legDur := leg.DurationSec / 60

		switch leg.Type {
		case types.LegTypeWalk:
			distStr := ""
			if leg.DistanceKm > 0 {
				distStr = fmt.Sprintf(", %.2f km", leg.DistanceKm)
			}
			fmt.Printf(" [%s] walk (%d min%s)\n", legDep, legDur, distStr)
			fmt.Printf("        └─ from : %s (wheelchair: %t, bikes: %t)\n", leg.FromStopName, IsAccessible(leg.Wheelchair), IsAccessible(leg.BikesAllowed))
			fmt.Printf("        └─ to   : %s (wheelchair: %t, bikes: %t)\n", leg.ToStopName, IsAccessible(leg.Wheelchair), IsAccessible(leg.BikesAllowed))
		case types.LegTypeBike:
			distStr := ""
			if leg.DistanceKm > 0 {
				distStr = fmt.Sprintf(", %.2f km @ 15 km/h", leg.DistanceKm)
			}
			fmt.Printf(" [%s] bike (%d min%s)\n", legDep, legDur, distStr)
			fmt.Printf("        └─ from : %s (wheelchair: %t, bikes: %t)\n", leg.FromStopName, IsAccessible(leg.Wheelchair), IsAccessible(leg.BikesAllowed))
			fmt.Printf("        └─ to   : %s (wheelchair: %t, bikes: %t)\n", leg.ToStopName, IsAccessible(leg.Wheelchair), IsAccessible(leg.BikesAllowed))
		case types.LegTypeTransit:
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
			pit := leg.PositionInTrain
			fmt.Println(pit)
			if pit.NbWagons > 0 {
				fmt.Printf("        │  (pos in train %d/%d) avg %d\n", int(pit.Wagon), int(pit.NbWagons), int(pit.PosAvg))
			}

			fmt.Printf(" [%s] └─ to   : %s (wheelchair: %t, bikes: %t)\n", legArr, leg.ToStopName, IsAccessible(leg.Wheelchair), IsAccessible(leg.BikesAllowed))
		}

		if i < len(trip.Legs)-1 {
			fmt.Println("        │")
		}
	}

	fmt.Println(strings.Repeat("═", 60))
}

func TripParsePositionInTrain(pit types.PositionInTrain) types.TripPositionInTrain {
	return types.TripPositionInTrain{
		NbWagons: pit.Wagons,
		Wagon:    pit.Pos,
		PosAvg:   pit.PositionAvg,
	}
}
