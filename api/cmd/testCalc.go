//go:build ignore

package main

import (
	"fmt"
	"log"
	"maps/api/internal/engine"
	"maps/api/internal/utils"
	"time"
)

func main() {
	startTime := time.Now()
	net, err := utils.LoadNetwork()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Take %d sec to load\n", int(time.Since(startTime).Seconds()))
	startTime = time.Now()

	engine := engine.StartEngine(*net)

	stop, exist := engine.SearchStopById("IDFM:64405")
	if !exist {
		fmt.Println("no exist")
	}
	fmt.Println(stop)

	fmt.Printf("Take %d sec to search id\n", int(time.Since(startTime).Seconds()))
	startTime = time.Now()

	stops := engine.SearchStopsByNeerCord(1.978210, 48.876648, 10)

	for _, s := range stops {
		fmt.Printf("%s %s %s\n", s.Name, s.SID, s.ParentStation)
	}

	fmt.Printf("Take %d sec to search list\n", int(time.Since(startTime).Seconds()))
	startTime = time.Now()

}
