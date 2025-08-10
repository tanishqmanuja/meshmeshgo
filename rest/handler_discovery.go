package rest

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"leguru.net/m/v2/meshmesh"
)

// @Id getDiscoveryProcedureState
// @Summary Get discovery procedure state
// @Tags    Discovery
// @Accept  json
// @Produce json
// @Success 200 {object} MeshDiscoveryState
// @Failure 400 {object} string
// @Router /api/discovery/state [get]
func (h Handler) getDiscoveryProcedureState(c *gin.Context) {
	discoveryState := MeshDiscoveryState{
		ID:        0,
		Status:    h.discoveryProcedure.StateString(),
		CurrentId: h.discoveryProcedure.CurrentDeviceId(),
		Repeat:    h.discoveryProcedure.CurrentRepeat(),
	}
	c.JSON(http.StatusOK, discoveryState)
}

func (h Handler) ctrlDiscoveryProcedure(c *gin.Context) {

	if h.discoveryProcedure.State() == meshmesh.DiscoveryProcedureStateDone || h.discoveryProcedure.State() == meshmesh.DiscoveryProcedureStateError {
		h.discoveryProcedure.Clear()
	}

	if h.discoveryProcedure.State() != meshmesh.DiscoveryProcedureStateRun {
		go h.discoveryProcedure.Run()
	}

	h.getDiscoveryProcedureState(c)
}

// @Id getNeighbors
// @Summary Get neighbors
// @Tags    Discovery
// @Accept  json
// @Produce json
// @Success 200 {array} MeshNeighbor
// @Failure 400 {object} string
// @Router /api/discovery/neighbors [get]
func (h Handler) getNeighbors(c *gin.Context) {

	if h.discoveryProcedure.Neighbors == nil {
		c.Header("Content-Range", "0-0/0")
		c.JSON(http.StatusOK, []MeshNeighbor{})
		return
	}

	jsonNeighbors := []MeshNeighbor{}
	for k, neighbor := range h.discoveryProcedure.Neighbors {
		jsonNeighbors = append(jsonNeighbors, MeshNeighbor{
			ID:      uint(k),
			Current: float32(neighbor.Current),
			Next:    float32(neighbor.Next),
			Delta:   float32(neighbor.Next - neighbor.Current),
		})
	}
	c.Header("Content-Range", fmt.Sprintf("%d-%d/%d", 0, len(jsonNeighbors), len(jsonNeighbors)))
	c.JSON(http.StatusOK, jsonNeighbors)
}
