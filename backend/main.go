package main

import (
	"maps/backend/cmd"
	"maps/backend/internal/api"
	"maps/backend/internal/engine"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		if os.Args[1] == "dataget" {
			cmd.RunFetchData()
			os.RemoveAll("data/raw")
			return
		}
	}
	e := engine.StartEngine()

	api.StartApi(e)
}
