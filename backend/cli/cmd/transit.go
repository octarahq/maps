package cmd

import (
	"fmt"
	"maps/backend/cli/cliutils"
	"maps/backend/internal/engine"
	"maps/backend/internal/types"
	"maps/backend/internal/utils"
	"time"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "transit",
	Short: "Search a trip",
	Long: `Search a trip
	
	maps transit --fromSID <SID> --toSID <SID> --arrivalAt <time>
	maps transit --from lat, long --to lat, long --departureAt <time>

	optionnal flag
	--withBike --accecibility-level [obligatory|(medium recommended)|none]
	`,
	Run: func(cmd *cobra.Command, args []string) {
		done := make(chan bool)
		go cliutils.ShowSpinner("Loading network", done)

		engine := engine.StartEngine()
		close(done)

		fromSID, _ := cmd.Flags().GetString("fromSID")
		toSID, _ := cmd.Flags().GetString("toSID")
		if fromSID == "" || toSID == "" {
			fmt.Println("need a start and an end")
			return
		}

		// IDFM:479556  IDFM:490906

		from, fromExist := engine.SearchStopById(fromSID)
		to, toExist := engine.SearchStopById(toSID)

		if !fromExist {
			fmt.Println("From stop is invalid")
			return
		}

		if !toExist {
			fmt.Println("End stop is invalid")
			return
		}

		fLong, fLat := from.Long, from.Lat
		tLong, tLat := to.Long, to.Lat

		arriveTimeStr, _ := cmd.Flags().GetString("arrivalAt")
		departureTimeStr, _ := cmd.Flags().GetString("departureAt")
		var targetTime = time.Now()
		var targetTimeType = types.TimeTypeDeparture
		var targetTimeError error

		if arriveTimeStr != "" {
			targetTime, targetTimeError = cliutils.TimeStrParse(arriveTimeStr)
			targetTimeType = types.TimeTypeArrival
		}

		if departureTimeStr != "" {
			targetTime, targetTimeError = cliutils.TimeStrParse(departureTimeStr)
			targetTimeType = types.TimeTypeDeparture
		}

		if targetTimeError != nil {
			fmt.Println(targetTimeError)
			return
		}

		timeTypeDisplay := "departure"
		if targetTimeType == types.TimeTypeArrival {
			timeTypeDisplay = "arrival"
		}

		withBike, _ := cmd.Flags().GetBool("withBike")

		fmt.Printf("Calculing for %s at %s\n", timeTypeDisplay, cliutils.TimeToString(targetTime))
		config := types.CalculatorConfig{
			TargetTime: targetTime,
			TimeType:   targetTimeType,

			WithBike:     withBike,
			AllowedModes: []types.RouteType{types.RouteTypeTrain, types.RouteTypeMetro, types.RouteTypeBus, types.RouteTypeTram},
		}

		calculatedTrips, err := engine.CalculateRouteCordsToCords(fLong, fLat, tLong, tLat, config)

		if err != nil {
			fmt.Println(err)
			return
		}

		for name, edges := range calculatedTrips {
			fmt.Println(name)
			trip := engine.EdgesToParsedTrip(edges, fLong, fLat, tLong, tLat, config)
			utils.PrintParsedTrip(trip)
		}
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().String("fromSID", "", "The stop id where you start")
	searchCmd.Flags().String("toSID", "", "The stop id where you finish")

	searchCmd.Flags().String("arrivalAt", "", "The time when you want to arrive")
	searchCmd.Flags().String("departureAt", "", "The time when you want to depart")

	searchCmd.Flags().Bool("withBike", false, "If you want to start your trip with a bike")
}
