package main

import (
	"maps/backend/internal/api"
	"maps/backend/internal/engine"
)

func main() {
	e := engine.StartEngine()

	api.StartApi(e)
}
