package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"leguru.net/m/v2/managerui"
	"leguru.net/m/v2/logger"
)

func serveStaticFiles(g *gin.Engine) {
	//g.Static("/manager", "./static")
	g.StaticFS("/manager", http.FS(managerui.Assets))
}

func StartRestServer(router Router, bindAddress string) {
	var g *gin.Engine
	if logger.IsTrace() {
		g = gin.Default()
	} else {
		g = gin.New()
	}
	gin.SetMode(gin.ReleaseMode)
	serveStaticFiles(g)
	router.Register(g)
	go g.Run(bindAddress)
}
