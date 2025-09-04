package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"leguru.net/m/v2/managerui"
)

func serveStaticFiles(g *gin.Engine) {
	//g.Static("/manager", "./static")
	g.StaticFS("/manager", http.FS(managerui.Assets))
}

func StartRestServer(router Router, bindAddress string) {
	g := gin.Default()
	serveStaticFiles(g)
	router.Register(g)
	go g.Run(bindAddress)
}
