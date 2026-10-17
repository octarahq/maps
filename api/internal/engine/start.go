package engine

import (
	"log"
	"maps/api/internal/types"
	"maps/api/internal/utils"
)

type Engine struct {
	Net   types.Network
	Graph types.GraphType
}

func StartEngine() Engine {
	var e Engine
	net, err := utils.LoadNetwork()
	if err != nil {
		log.Fatal(err)
	}
	e.Net = *net

	e.Graph = utils.BuildGraph(net)

	return e
}
