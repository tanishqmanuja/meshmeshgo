package rest

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h Handler) getFirmware(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	dev, err := h.network.GetNodeDevice(int64(id))
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

func (h Handler) updateFirmware(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	dev, err := h.network.GetNodeDevice(int64(id))
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
