package rest

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"leguru.net/m/v2/graph"
	"leguru.net/m/v2/logger"
	"leguru.net/m/v2/utils"
)

// @Id getEsphomeServers
// @Summary Get esphome servers
// @Tags    Esphome
// @Accept  json
// @Produce json
// @Success 200 {array} EsphomeServer
// @Failure 400 {object} string
// @Router /api/esphome/servers [get]
func (h Handler) getEsphomeServers(c *gin.Context) {
	jsonServers := make([]EsphomeServer, 0)
	for _, server := range h.esphomeServers.Servers {
		jsonServers = append(jsonServers, EsphomeServer{
			ID:      uint(server.Address),
			Address: utils.FmtNodeIdHass(int64(server.Address)) + ":6053",
			Clients: len(server.Clients),
		})
	}
	c.Header("Content-Range", fmt.Sprintf("%d-%d/%d", 0, len(jsonServers), len(jsonServers)))
	c.JSON(http.StatusOK, jsonServers)
}

// @Id getEsphomeConnections
// @Summary Get esphome connections
// @Tags    Esphome
// @Accept  json
// @Produce json
// @Success 200 {array} EsphomeClient
// @Failure 400 {object} string
// @Router /api/esphome/connections [get]
func (h Handler) getEsphomeConnections(c *gin.Context) {
	index := uint(1)

	stats := h.esphomeServers.Stats()

	jsonClients := make([]EsphomeClient, 0)
	for nodeId, connection := range stats.Connections {

		network := graph.GetMainNetwork()
		dev, err := network.GetNodeDevice(int64(nodeId))
		if err != nil {
			logger.WithField("id", nodeId).Error("Node not found for esphome connection")
			continue
		}

		jsonClients = append(jsonClients, EsphomeClient{
			ID:       index,
			Address:  utils.FmtNodeIdHass(int64(nodeId)),
			Tag:      dev.Device().Tag(),
			Active:   connection.IsActive(),
			Handle:   int(connection.GetLastHandle()),
			Sent:     connection.BytesOut(),
			Received: connection.BytesIn(),
			Duration: connection.TimeSinceLastConnection().String(),
			Started:  connection.LastConnectionDuration().String(),
		})
	}

	c.Header("Content-Range", fmt.Sprintf("%d-%d/%d", 0, len(jsonClients), len(jsonClients)))
	c.JSON(http.StatusOK, jsonClients)
}
