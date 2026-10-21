package engine

import (
	"log"
	"maps/api/internal/types"
	"maps/api/internal/utils"
)

type Engine struct {
	Net          types.Network
	Graph        types.GraphType
	ReverseGraph types.GraphType
	Extras       types.ExtraData
}

func StartEngine() Engine {
	var e Engine
	net, err := utils.LoadNetwork()
	if err != nil {
		log.Fatal(err)
	}
	e.Net = *net

	e.Graph = utils.BuildGraph(net)
	e.ReverseGraph = utils.BuildReverseGraph(e.Graph)

	extra, err := utils.LoadExtra()
	if err == nil && extra != nil {
		e.Extras = *extra
	}

	return e
}
