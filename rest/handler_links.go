package rest

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"leguru.net/m/v2/graph"
)

// @Id getLinks
// @Summary Get links
// @Tags    Links
// @Accept  json
// @Produce json
// @Param   login body GetListRequest true "Get list request"
// @Success 200 {array} MeshLink
// @Failure 400 {string} string
// @Router /api/links [get]
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

	network := graph.GetMainNetwork()
	links := network.WeightedEdges()
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

	sort.Slice(jsonLinks, func(i, j int) bool {
		switch p.SortType {
		case sortTypeAsc:
			switch p.SortBy {
			case sortFieldTypeID:
				return jsonLinks[i].ID < jsonLinks[j].ID
			case sortFieldTypeHExId:
				return jsonLinks[i].ID < jsonLinks[j].ID
			case sortFieldTypeFrom:
				return jsonLinks[i].From < jsonLinks[j].From
			case sortFieldTypeTo:
				return jsonLinks[i].To < jsonLinks[j].To
			case sortFieldTypeWeight:
				return jsonLinks[i].Weight < jsonLinks[j].Weight
			}
			return jsonLinks[i].ID < jsonLinks[j].ID
		case sortTypeDesc:
			switch p.SortBy {
			case sortFieldTypeID:
				return jsonLinks[i].ID > jsonLinks[j].ID
			case sortFieldTypeHExId:
				return jsonLinks[i].ID > jsonLinks[j].ID
			case sortFieldTypeFrom:
				return jsonLinks[i].From > jsonLinks[j].From
			case sortFieldTypeTo:
				return jsonLinks[i].To > jsonLinks[j].To
			case sortFieldTypeWeight:
				return jsonLinks[i].Weight > jsonLinks[j].Weight
			}
			return jsonLinks[i].ID > jsonLinks[j].ID
		}
		return false
	})

	c.Header("Content-Range", fmt.Sprintf("%d-%d/%d", 0, len(jsonLinks), len(jsonLinks)))
	c.JSON(http.StatusOK, jsonLinks)
}

// @Id getOneLink
// @Summary Get one link
// @Tags    Links
// @Accept  json
// @Produce json
// @Param   id path int true "Link ID"
// @Success 200 {object} MeshLink
// @Failure 400 {string} string
// @Router /api/links/{id} [get]
func (h Handler) getOneLink(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	fromID := uint(id) & 0x00FFFFFF
	toID := uint(id) >> 24

	network := graph.GetMainNetwork()
	edge := network.WeightedEdge(int64(fromID), int64(toID))
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

// @Id createLink
// @Summary Create link
// @Tags    Links
// @Accept  json
// @Produce json
// @Param   link body bool true "Create link request"
// @Success 200 {object} MeshLink
// @Failure 400 {object} string
// @Router /api/links [post]
func (h Handler) createLink(c *gin.Context) {
	_ = c.AbortWithError(http.StatusBadRequest, errors.New("not implemented"))
}

// @Id updateLink
// @Summary Update link
// @Tags    Links
// @Accept  json
// @Produce json
// @Param   id path integer true "Mixed FromTo ID"
// @Param   link body UpdateLinkRequest true "Update link request"
// @Success 200 {object} MeshLink
// @Failure 400 {object} string
// @Router /api/links/{id} [put]
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

	network := graph.GetMainNetwork()
	edge := network.WeightedEdge(int64(fromID), int64(toID))
	if edge == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Link not found"})
		return
	}

	network.ChangeEdgeWeight(int64(fromID), int64(toID), float64(req.Weight), float64(req.Weight))
	graph.NotifyMainNetworkChanged()

	jsonLink := MeshLink{
		ID:     uint(fromID) + uint(toID)<<24,
		From:   uint(fromID),
		To:     uint(toID),
		Weight: float32(req.Weight),
	}

	c.JSON(http.StatusOK, jsonLink)
}

// @Id deleteLink
// @Summary Delete link
// @Tags    Links
// @Accept  json
// @Produce json
// @Param   id path integer true "Mixed FromTo ID"
// @Success 200 {object} MeshLink
// @Failure 400 {string} string
// @Router /api/links/{id} [delete]
func (h Handler) deleteLink(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	fromID := uint(id) & 0x00FFFFFF
	toID := uint(id) >> 24

	network := graph.GetMainNetwork()
	network.RemoveEdge(int64(fromID), int64(toID))
	graph.NotifyMainNetworkChanged()

	jsonLink := MeshLink{
		ID:     uint(fromID) + uint(toID)<<24,
		From:   uint(fromID),
		To:     uint(toID),
		Weight: 0,
	}

	c.JSON(http.StatusOK, jsonLink)
}
