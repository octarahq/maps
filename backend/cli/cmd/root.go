package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "maps",
	Short: "Never get lost in the IDFM network !",
	Long:  `Search a trip with detailed steps on the IDFM network https://iledefrance-mobilites.fr`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {

}
