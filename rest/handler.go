package rest

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"leguru.net/m/v2/graph"
	mm "leguru.net/m/v2/meshmesh"
)

type Handler struct {
	serialConn              *mm.SerialConnection
	discoveryProcedure      *mm.DiscoveryProcedure
	firmwareUploadProcedure *mm.FirmwareUploadProcedure
	esphomeServers          *mm.MultiServerApi
}

func smartInteger(v any) int64 {
	switch v := v.(type) {
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return -1
		}
		return i
	case int:
		return int64(v)
	case float64:
		return int64(v)
	}

	return -1
}

func routeFrontend(c *gin.Context) {
	c.Status(http.StatusFound)
	c.Writer.Header().Set("Location", "/manager")
}

func NewHandler(serialConn *mm.SerialConnection, network *graph.Network, esphomeServers *mm.MultiServerApi) Handler {
	return Handler{
		serialConn:              serialConn,
		discoveryProcedure:      mm.NewDiscoveryProcedure(serialConn, nil, network.LocalDeviceId()),
		firmwareUploadProcedure: nil,
		esphomeServers:          esphomeServers,
	}
}
