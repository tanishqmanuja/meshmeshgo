package rest

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h Handler) getLinks(c *gin.Context) {
	var req GetListRequest
	err := c.ShouldBindQuery(&req)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	p := req.toGetListParams()

	filter_to := smartInteger(p.Filter["to"])
	filter_from := smartInteger(p.Filter["from"])
	filter_any := smartInteger(p.Filter["any"])

	links := h.network.WeightedEdges()
	jsonLinks := make([]MeshLink, 0, links.Len())
	for links.Next() {
		edge := links.WeightedEdge()
		fromID := edge.From().ID()
		toID := edge.To().ID()

		if ((filter_to != -1 && filter_to != toID) && (filter_from != -1 && filter_from != fromID)) || (filter_any != -1 && (filter_any != fromID && filter_any != toID)) {
			continue
		}

		jsonLinks = append(jsonLinks, MeshLink{
			ID:     uint(fromID) + uint(toID)<<24,
			From:   uint(fromID),
			To:     uint(toID),
			Weight: float32(edge.Weight()),
		})
	}

	c.Header("Content-Range", fmt.Sprintf("%d-%d/%d", 0, len(jsonLinks), len(jsonLinks)))
	c.JSON(http.StatusOK, jsonLinks)
}

func (h Handler) getOneLink(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	fromID := uint(id) & 0x00FFFFFF
	toID := uint(id) >> 24

	edge := h.network.WeightedEdge(int64(fromID), int64(toID))
	if edge == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Link not found"})
		return
	}

	jsonLink := MeshLink{
		ID:     uint(fromID) + uint(toID)<<24,
		From:   uint(fromID),
		To:     uint(toID),
		Weight: float32(edge.Weight()),
	}

	c.JSON(http.StatusOK, jsonLink)
}

func (h Handler) createLink(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Hello, World!"})
}

func (h Handler) updateLink(c *gin.Context) {
	idStr := c.Param("id")
	fmt.Println(c.Param("data"))
	fmt.Println(c.Param("previousData"))

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	req := UpdateLinkRequest{}
	err = c.ShouldBindJSON(&req)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	fromID := uint(id) & 0x00FFFFFF
	toID := uint(id) >> 24

	edge := h.network.WeightedEdge(int64(fromID), int64(toID))
	if edge == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Link not found"})
		return
	}

	h.network.ChangeEdgeWeight(int64(fromID), int64(toID), float64(req.Weight), float64(req.Weight))
	h.network.NotifyNetworkChanged()

	jsonLink := MeshLink{
		ID:     uint(fromID) + uint(toID)<<24,
		From:   uint(fromID),
		To:     uint(toID),
		Weight: float32(req.Weight),
	}

	c.JSON(http.StatusOK, jsonLink)
}

func (h Handler) deleteLink(c *gin.Context) {
}
