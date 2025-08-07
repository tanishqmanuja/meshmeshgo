package rest

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Router interface {
	Register(g gin.IRouter)
}

type router struct {
	handler Handler
}

func NewRouter(handler Handler) Router {
	return &router{handler: handler}
}

func (s router) Register(g gin.IRouter) {
	h := s.handler

	g.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "FETCH"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"*"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	r := g.Group("/api/v1")
	nodesGroup := r.Group("/nodes")
	{
		nodesGroup.GET("", h.getNodes)
		nodesGroup.GET("/:id", h.getOneNode)
		nodesGroup.POST("", h.createNode)
		nodesGroup.PUT("/:id", h.updateNode)
		nodesGroup.DELETE("/:id", h.deleteNode)
	}

	linksGroup := r.Group("/links")
	{
		linksGroup.GET("", h.getLinks)
		linksGroup.GET("/:id", h.getOneLink)
		linksGroup.POST("", h.createLink)
		linksGroup.PUT("/:id", h.updateLink)
		linksGroup.DELETE("/:id", h.deleteLink)
	}

	neighborsGroup := r.Group("/neighbors")
	{
		neighborsGroup.GET("", h.getNeighbors)
		neighborsGroup.GET("/discovery/:id", h.getDiscoveryProcedureState)
		neighborsGroup.POST("/discovery", h.ctrlDiscoveryProcedure)
	}
}
