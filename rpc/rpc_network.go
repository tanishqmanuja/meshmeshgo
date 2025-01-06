package rpc

import (
	"context"

	"leguru.net/m/v2/graph"
	"leguru.net/m/v2/rpc/meshmesh"
)

func (s *Server) NetworkNodes(_ context.Context, req *meshmesh.NetworkNodesRequest) (*meshmesh.NetworkNodesReply, error) {
	nodes := s.network.Nodes()
	device := make([]*meshmesh.NetworkNode, nodes.Len())
	i := 0
	for nodes.Next() {
		dev := nodes.Node().(*graph.Device)
		device[i] = &meshmesh.NetworkNode{
			Id:    uint32(dev.ID()),
			Tag:   string(dev.Tag()),
			Inuse: dev.InUse(),
		}
		i += 1
	}
	return &meshmesh.NetworkNodesReply{Nodes: device}, nil
}

func (s *Server) NetworkEdges(_ context.Context, req *meshmesh.NetworkEdgesRequest) (*meshmesh.NetworkEdgesReply, error) {
	edges := s.network.WeightedEdges()
	_edges := make([]*meshmesh.NetworkEdge, edges.Len())
	i := 0
	for edges.Next() {
		edge := edges.WeightedEdge()
		_edges[i] = &meshmesh.NetworkEdge{
			From:   uint32(edge.From().ID()),
			To:     uint32(edge.To().ID()),
			Weight: float32(edge.Weight()),
		}
		i += 1
	}
	return &meshmesh.NetworkEdgesReply{Edges: _edges}, nil
}
