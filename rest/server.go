package rest

import (
	"net/http"

	"leguru.net/m/v2/managerui"

	"github.com/gin-gonic/gin"
)

func serveStaticFiles(g *gin.Engine) {
	//g.Static("/manager", "./static")
	g.StaticFS("/manager", http.FS(managerui.Assets))
}

func StartRestServer(router Router) {
	g := gin.Default()
	serveStaticFiles(g)
	router.Register(g)
	go g.Run(":4002")
}
