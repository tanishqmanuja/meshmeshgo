package graph

import (
	"fmt"
	"math"

	"github.com/charmbracelet/log"
	"github.com/sirupsen/logrus"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/path"
	"gonum.org/v1/gonum/graph/simple"
)

type GraphPathConnection struct {
	Target uint32
	Port   uint16
	Path   []byte
	Handle uint16
}

type MeshNodeAttrs struct {
	isInUse      bool
	isDiscovered bool
	tag          string
}

type MeshEdgeAttrs struct {
	wieght float32
}

type MeshNode struct {
	id int64
}

type GraphPath struct {
	SourceNode int64
	attrs      map[int64]MeshNodeAttrs
	edgeAttrs  map[int64]MeshEdgeAttrs
	Graph      *simple.WeightedDirectedGraph
}

func (n MeshNode) ID() int64 {
	return n.id
}

func NewMeshNode(id int64) MeshNode {
	return MeshNode{id: id}
}

func (g *GraphPath) GetPath(to int64) ([]int64, float64, error) {
	node := g.Graph.Node(to)
	if node == nil {
		return nil, 0, fmt.Errorf("node is 0x%06X is not prsent in graph", to)
	}
	meshNode := node.(MeshNode)
	if !g.attrs[meshNode.ID()].isInUse {
		return nil, 0, fmt.Errorf("node is 0x%06X is not active", to)
	}
	allShortest := path.DijkstraAllPaths(g.Graph)
	allBetween, weight := allShortest.AllBetween(g.SourceNode, to)
	if len(allBetween) == 0 {
		return nil, 0, fmt.Errorf("no path found between 0x%06X and 0x%06X", g.SourceNode, to)
	}
	logrus.WithFields(logrus.Fields{"length": len(allBetween[0]), "weight": weight}).
		Debug(fmt.Sprintf("Get path from 0x%06X to 0x%06X", g.SourceNode, to))

	nodes := allBetween[0]
	path := make([]int64, len(nodes))
	for i, item := range nodes {
		path[i] = item.ID()
	}

	return path, 0, nil
}

func (g *GraphPath) NodeExists(id int64) bool {
	if _, ok := g.attrs[id]; ok {
		return true
	} else {
		return false
	}
}

func (g *GraphPath) NodeIsInUse(id int64) bool {
	var inUse bool
	if entry, ok := g.attrs[id]; ok {
		inUse = entry.isInUse
	}
	return inUse
}

func (g *GraphPath) SetNodeIsInUse(id int64, isInUse bool) {
	if entry, ok := g.attrs[id]; ok {
		entry.isInUse = isInUse
		g.attrs[id] = entry
	}
}

func (g *GraphPath) NodeIsDiscovered(id int64) bool {
	var inDiscovered bool
	if entry, ok := g.attrs[id]; ok {
		inDiscovered = entry.isDiscovered
	}
	return inDiscovered
}

func (g *GraphPath) SetNodeIsDiscovered(id int64, isDiscovered bool) {
	if entry, ok := g.attrs[id]; ok {
		entry.isDiscovered = isDiscovered
		g.attrs[id] = entry
	}
}

func (g *GraphPath) NodeTag(id int64) string {
	var tag string
	if entry, ok := g.attrs[id]; ok {
		tag = entry.tag
	}
	return tag
}

func (g *GraphPath) SetNodeTag(id int64, tag string) {
	if entry, ok := g.attrs[id]; ok {
		entry.tag = tag
		g.attrs[id] = entry
	}
}

func (g *GraphPath) EdgeWeight(from int64, to int64) float32 {
	var weight float32
	id := from + (to << 24)
	if entry, ok := g.edgeAttrs[id]; ok {
		weight = entry.wieght
	}
	return weight
}

func (g *GraphPath) SetEdgeWeight(from int64, to int64, weight float32) {
	id := from + (to << 24)
	if entry, ok := g.edgeAttrs[id]; ok {
		entry.wieght = weight
		g.edgeAttrs[id] = entry
	}
}

func (g *GraphPath) GetAllInUse() []int64 {
	var res []int64
	for id, a := range g.attrs {
		if a.isInUse {
			res = append(res, id)
		}
	}
	return res
}

func (g *GraphPath) AddNode(id int64) error {
	n := NewMeshNode(id)
	g.Graph.AddNode(n)
	g.attrs[n.ID()] = MeshNodeAttrs{isInUse: true}
	return nil
}

func (g *GraphPath) AddNodeIfNotExists(id int64) graph.Node {
	n := g.Graph.Node(id)
	if n == nil {
		n = NewMeshNode(id)
		g.Graph.AddNode(n)
		g.attrs[n.ID()] = MeshNodeAttrs{isInUse: true}
	}
	return n
}

func (g *GraphPath) ChangeEdgeWeight(fromId int64, toId int64, weightFrom float64, weightTo float64) {
	fromNode := g.Graph.Node(fromId)
	toNode := g.AddNodeIfNotExists(toId)

	if !g.Graph.HasEdgeFromTo(fromId, toId) {
		edgeTo := g.Graph.NewWeightedEdge(fromNode, toNode, weightTo)
		g.Graph.SetWeightedEdge(edgeTo)
	} else {
		edgeTo := g.Graph.WeightedEdge(fromId, toId)
		oldWeightTo := edgeTo.Weight()
		newWeightTo := (oldWeightTo + weightTo) / 2
		newEdgeTo := g.Graph.NewWeightedEdge(fromNode, toNode, newWeightTo)
		g.Graph.SetWeightedEdge(newEdgeTo)
	}

	if !g.Graph.HasEdgeFromTo(toId, fromId) {
		edgeFrom := g.Graph.NewWeightedEdge(toNode, fromNode, weightFrom)
		g.Graph.SetWeightedEdge(edgeFrom)
	} else {
		edgeFrom := g.Graph.WeightedEdge(toId, fromId)
		oldWeightFrom := edgeFrom.Weight()
		newWeightFrom := (oldWeightFrom + weightFrom) / 2
		newEdgeFrom := g.Graph.NewWeightedEdge(toNode, fromNode, newWeightFrom)
		g.Graph.SetWeightedEdge(newEdgeFrom)
	}
}

func NewGraphPath(sourcenode int64) (*GraphPath, error) {
	graph := GraphPath{SourceNode: sourcenode, attrs: make(map[int64]MeshNodeAttrs), edgeAttrs: make(map[int64]MeshEdgeAttrs)}
	graph.Graph = simple.NewWeightedDirectedGraph(0, math.Inf(1))
	graph.AddNode(sourcenode)
	return &graph, nil
}

func NewGraphPathFromFile(filename string, sourcenode int64) (*GraphPath, error) {
	graph := GraphPath{SourceNode: sourcenode, attrs: make(map[int64]MeshNodeAttrs), edgeAttrs: make(map[int64]MeshEdgeAttrs)}
	//graph.readGraphXml()
	err := graph.readGraph()
	if err != nil {
		log.Error(err)
	}
	return &graph, nil
}
