package rest

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"leguru.net/m/v2/graph"
)

// @Id getFirmware
// @Summary Get firmware
// @Tags    Firmware
// @Accept  json
// @Produce json
// @Param   id path string true "Firmware ID"
// @Success 200 {object} MeshFirmware
// @Failure 400 {object} string
// @Router /api/firmware/{id} [get]
func (h *Handler) getFirmware(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	network := graph.GetMainNetwork()
	dev, err := network.GetNodeDevice(int64(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Node not found: " + err.Error()})
		return
	}

	jsonFirmware := MeshFirmware{
		ID:       dev.ID(),
		Status:   "pending",
		Filename: "firmware.bin",
		Size:     1024,
		Progress: 50,
	}
	c.JSON(http.StatusOK, jsonFirmware)
}

// @Id updateFirmware
// @Summary Update firmware
// @Tags    Firmware
// @Accept  json
// @Produce json
// @Param   id path string true "Firmware ID"
// @Param   firmware body UpdateFirmwareRequest true "Update firmware request"
// @Success 200 {object} MeshFirmware
// @Failure 400 {object} string
// @Router /api/firmware/{id} [put]
func (h *Handler) updateFirmware(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	req := UpdateFirmwareRequest{}
	err = c.ShouldBindJSON(&req)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	network := graph.GetMainNetwork()
	dev, err := network.GetNodeDevice(int64(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Node not found: " + err.Error()})
		return
	}

	jsonFirmware := MeshFirmware{
		ID:       dev.ID(),
		Status:   "pending",
		Filename: req.Filename,
		Size:     1024,
		Progress: 50,
	}

	c.JSON(http.StatusOK, jsonFirmware)
}
