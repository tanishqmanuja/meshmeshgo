package rest

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
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

// @Id getEsphomeClients
// @Summary Get esphome clients
// @Tags    Esphome
// @Accept  json
// @Produce json
// @Success 200 {array} EsphomeClient
// @Failure 400 {object} string
// @Router /api/esphome/clients [get]
func (h Handler) getEsphomeClients(c *gin.Context) {
	index := uint(1)

	jsonClients := make([]EsphomeClient, 0)
	for _, server := range h.esphomeServers.Servers {

		dev, err := h.network.GetNodeDevice(int64(server.Address))
		if err != nil {
			logger.WithField("id", server.Address).Error("Node not found for esphome client")
			continue
		}

		for _, client := range server.Clients {
			jsonClients = append(jsonClients, EsphomeClient{
				ID:       index,
				Address:  utils.FmtNodeIdHass(int64(server.Address)),
				Tag:      dev.Device().Tag(),
				Active:   client.Stats.IsActive(),
				Handle:   int(client.Stats.GetLastHandle()),
				Sent:     client.Stats.BytesOut(),
				Received: client.Stats.BytesIn(),
				Duration: client.Stats.TimeSinceLastConnection().String(),
				Started:  client.Stats.LastConnectionDuration().String(),
			})
		}
	}
	c.Header("Content-Range", fmt.Sprintf("%d-%d/%d", 0, len(jsonClients), len(jsonClients)))
	c.JSON(http.StatusOK, jsonClients)
}
