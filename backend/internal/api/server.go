package api

import (
	"maps/backend/internal/api/pages"
	"maps/backend/internal/api/routes"
	"maps/backend/internal/engine"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func StartApi(engine engine.Engine) {
	router := gin.Default()
	router.Use(gin.Logger())
	router.Use(cors.Default())

	pages.Register(router, engine)

	api := router.Group("/api")
	{
		api.Use(func(ctx *gin.Context) {
			ctx.Next()
		})

		for _, route := range routes.ApiRoutes {
			api.Handle(route.Method, route.Path, func(ctx *gin.Context) {
				route.Handler(ctx, engine)
			})
		}
	}

	router.Run(":4021")
}
