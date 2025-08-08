package rest

import "encoding/json"

type GetListRequest struct {
	Filter map[string]any `form:"filter"`
	Range  string         `form:"range"`
	Sort   string         `form:"sort"`
}

type CreateNodeRequest struct {
	ID    uint   `json:"id"`
	Tag   string `json:"tag"`
	InUse bool   `json:"in_use"`
}

type UpdateNodeRequest struct {
	Tag     string `json:"tag"`
	InUse   bool   `json:"in_use"`
	DevTag  string `json:"dev_tag"`
	Channel int8   `json:"channel"`
	TxPower int8   `json:"tx_power"`
}

type MeshNode struct {
	ID       uint   `json:"id"`
	Tag      string `json:"tag"`
	InUse    bool   `json:"in_use"`
	Path     string `json:"path"`
	Revision string `json:"revision"`
	Error    string `json:"error"`
	DevTag   string `json:"dev_tag"`
	Channel  int8   `json:"channel"`
	TxPower  int8   `json:"tx_power"`
	Groups   int    `json:"groups"`
	Binded   int    `json:"binded"`
	Flags    int    `json:"flags"`
}

type UpdateLinkRequest struct {
	ID     uint    `json:"id"`
	Weight float32 `json:"weight"`
}

type MeshLink struct {
	ID     uint    `json:"id"`
	From   uint    `json:"from"`
	To     uint    `json:"to"`
	Weight float32 `json:"weight"`
}

type MeshNeighbor struct {
	ID      uint    `json:"id"`
	Address string  `json:"address"`
	Current float32 `json:"current"`
	Next    float32 `json:"next"`
	Delta   float32 `json:"delta"`
}

type MeshDiscoveryState struct {
	ID        int64  `json:"id"`
	Status    string `json:"status"`
	CurrentId int64  `json:"current_id"`
	Repeat    int    `json:"repeat"`
}

var acceptFilters = map[string]struct{}{
	"from": {},
	"to":   {},
	"any":  {},
}

type GetListParams struct {
	Filter           map[string]interface{}
	Limit, Offset    int
	SortBy, SortType string
}

func (r GetListRequest) toGetListParams() GetListParams {
	acceptFiltersParam := make(map[string]any)
	for k, v := range r.Filter {
		if _, ok := acceptFilters[k]; ok {
			acceptFiltersParam[k] = v
		}
	}

	p := GetListParams{
		Filter:   acceptFiltersParam,
		Limit:    10,
		Offset:   0,
		SortBy:   "id",
		SortType: "ASC",
	}

	if r.Range != "" {
		var queryRange []int
		_ = json.Unmarshal([]byte(r.Range), &queryRange)
		if len(queryRange) == 2 {
			p.Offset, p.Limit = queryRange[0], queryRange[1]
		}
	}

	if r.Sort != "" {
		var querySort []string
		_ = json.Unmarshal([]byte(r.Sort), &querySort)
		if len(querySort) == 2 {
			p.SortBy, p.SortType = querySort[0], querySort[1]
		}
	}

	return p
}
