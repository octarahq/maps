package routes

import (
	"maps/backend/internal/engine"

	"github.com/gin-gonic/gin"
)

type ApiRoute struct {
	Method  string
	Path    string
	Handler func(c *gin.Context, e engine.Engine)
}

var ApiRoutes = []ApiRoute{}

func Register(route ApiRoute) {
	ApiRoutes = append(ApiRoutes, route)
}
