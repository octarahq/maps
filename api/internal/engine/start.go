package engine

import "maps/api/internal/types"

type Engine struct {
	Net types.Network
}

func StartEngine(net types.Network) Engine {
	var e Engine
	e.Net = net
	return e
}
