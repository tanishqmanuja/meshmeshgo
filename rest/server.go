package rest

import "github.com/gin-gonic/gin"

func serveStaticFiles(g *gin.Engine) {
	g.Static("/static", "./static")
}

func StartRestServer(router Router) {
	g := gin.Default()
	serveStaticFiles(g)
	router.Register(g)
	go g.Run(":4002")
}
