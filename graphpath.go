package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/path"
	"gonum.org/v1/gonum/graph/simple"
)

type XmlGraphml struct {
	XMLName xml.Name   `xml:"graphml"`
	Keys    []XmlKey   `xml:"key"`
	Graphs  []XmlGraph `xml:"graph"`
}

type XmlKey struct {
	XMLName  xml.Name `xml:"key"`
	Id       string   `xml:"id,attr"`
	For      string   `xml:"for,attr"`
	AttrName string   `xml:"attr.name,attr"`
	AttrType string   `xml:"attr.type,attr"`
}

type XmlGraph struct {
	XMLName     xml.Name  `xml:"graph"`
	EdgeDefault string    `xml:"edgedefault,attr"`
	Nodes       []XmlNode `xml:"node"`
	Edges       []XmlEdge `xml:"edge"`
}

type XmlNode struct {
	XMLName xml.Name  `xml:"node"`
	Id      string    `xml:"id,attr"`
	Data    []XmlData `xml:"data"`
}

type XmlEdge struct {
	XMLName xml.Name  `xml:"edge"`
	Source  string    `xml:"source,attr"`
	Target  string    `xml:"target,attr"`
	Data    []XmlData `xml:"data"`
}

type XmlData struct {
	XMLName xml.Name `xml:"data"`
	Text    string   `xml:",chardata"`
	Key     string   `xml:"key,attr"`
}

type GraphPathConnection struct {
	Target uint32
	Port   uint16
	Path   []byte
	Handle uint16
}

type GraphPath struct {
	SourceNode int64
	Graph      *simple.WeightedDirectedGraph
}

type MeshGraph struct {
	SourceNodeId int64
	simple.WeightedDirectedGraph
}

type MeshNode struct {
	id         int64
	inUse      bool
	Discovered bool
}

func (n MeshNode) ID() int64 {
	return n.id
}

func (n MeshNode) GetInUse() bool {
	return n.inUse
}

func NewMeshNode(id int64) MeshNode {
	return MeshNode{id: id, inUse: true}
}

func parseNodeId(id string) (int64, error) {
	if len(id) < 3 {
		return 0, errors.New("invalid id string")
	}
	return strconv.ParseInt(id, 0, 32)
}

func keyForAttribute(data []XmlKey, attribute string) string {
	for i := range data {
		if data[i].AttrName == attribute {
			return data[i].Id
		}
	}
	return ""
}

func attribteOfNode(data []XmlData, key string) string {
	for i := range data {
		if data[i].Key == key {
			return data[i].Text
		}
	}
	return ""
}

func boolAttribteOfNode(data []XmlData, key string) (bool, error) {
	ret, err := strconv.ParseBool(attribteOfNode(data, key))
	return ret, err
}

func (gpath *GraphPath) readGraphXml() {
	xmlFile, err := os.Open("meshmesh.graphml")
	if err != nil {
		fmt.Println("ReadGraphXml", err)
	}

	defer xmlFile.Close()
	byteValue, _ := io.ReadAll(xmlFile)
	var xmlgraphml XmlGraphml
	xml.Unmarshal(byteValue, &xmlgraphml)
	inusekey := keyForAttribute(xmlgraphml.Keys, "inuse")
	if inusekey == "" {
		log.Error("Missing inuse field in graph")
		return
	}
	for i, graph := range xmlgraphml.Graphs {
		if i == 0 {
			gpath.Graph = simple.NewWeightedDirectedGraph(0, math.Inf(1))
			for _, node := range graph.Nodes {
				node_id, err := parseNodeId(node.Id)
				if err != nil {
					log.Println("ReadGraphXml", err)
					continue
				}
				n := NewMeshNode(node_id)
				n.inUse, err = boolAttribteOfNode(node.Data, inusekey)
				if err != nil {
					log.WithField("node", node.Id).Error("Mssing inuse field in node")
				}
				gpath.Graph.AddNode(n)
			}

			for _, edge := range graph.Edges {
				src_id, err := parseNodeId(edge.Source)
				if err != nil {
					log.Println("ReadGraphXml", err)
					continue
				}
				dst_id, err := parseNodeId(edge.Target)
				if err != nil {
					log.Println("ReadGraphXml", err)
					continue
				}
				weight, err := strconv.ParseFloat(edge.Data[0].Text, 32)
				if err != nil {
					log.Println("ReadGraphXml", err)
					continue
				}

				edge := gpath.Graph.NewWeightedEdge(gpath.Graph.Node(src_id), gpath.Graph.Node(dst_id), weight)
				gpath.Graph.SetWeightedEdge(edge)

			}
		}
	}

	log.WithFields(logrus.Fields{"nodes": gpath.Graph.Nodes().Len(), "edges": gpath.Graph.Edges().Len()}).Info("Readed graphml from file")
}

func (g *GraphPath) GetPath(to int64) ([]int64, float64, error) {
	node := g.Graph.Node(to)
	if node == nil {
		return nil, 0, fmt.Errorf("node is 0x%06X is not prsent in graph", to)
	}
	meshNode := node.(MeshNode)
	if !meshNode.inUse {
		return nil, 0, fmt.Errorf("node is 0x%06X is not active", to)
	}
	allShortest := path.DijkstraAllPaths(g.Graph)
	allBetween, weight := allShortest.AllBetween(g.SourceNode, to)
	if len(allBetween) == 0 {
		return nil, 0, fmt.Errorf("no path found between 0x%06X and 0x%06X", g.SourceNode, to)
	}
	log.WithFields(logrus.Fields{"length": len(allBetween[0]), "weight": weight}).
		Info(fmt.Sprintf("Get path from 0x%06X to 0x%06X", g.SourceNode, to))

	nodes := allBetween[0]
	path := make([]int64, len(nodes))
	for i, item := range nodes {
		path[i] = item.ID()
	}

	return path, 0, nil
}

func (g *GraphPath) AddNode(id int64) error {
	n := NewMeshNode(id)
	g.Graph.AddNode(n)
	return nil
}

func (g *GraphPath) AddNodeIfNotExists(id int64) graph.Node {
	n := g.Graph.Node(id)
	if n == nil {
		n = NewMeshNode(id)
		g.Graph.AddNode(n)
	}
	return n
}

func (g *GraphPath) ChangeEdgetWeight(fromId int64, toId int64, weightFrom float64, weightTo float64) {
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
	graph := GraphPath{SourceNode: sourcenode}
	graph.Graph = simple.NewWeightedDirectedGraph(0, math.Inf(1))
	graph.AddNode(sourcenode)
	return &graph, nil
}

func NewGraphPathFromFile(filename string, sourcenode int64) (*GraphPath, error) {
	graph := GraphPath{SourceNode: sourcenode}
	graph.readGraphXml()
	return &graph, nil
}
