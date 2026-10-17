//go:build ignore

package main

import (
	"flag"
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

	query := flag.String("search", "", "")
	flag.Parse()

	res := engine.SearchStopsByName(*query)

	if len(res) > 10 {
		res = res[0:10]
	}
	for _, s := range res {
		fmt.Printf("%s %s\n", s.Name, s.SID)
	}

	fmt.Printf("Take %d sec to search\n", int(time.Since(startTime).Seconds()))
	startTime = time.Now()

	stop, exist := engine.SearchStopById("IDFM:64405")
	if !exist {
		fmt.Println("no exist")
	}
	fmt.Println(stop.Name)

	fmt.Printf("Take %d sec to search id\n", int(time.Since(startTime).Seconds()))
	startTime = time.Now()

}
