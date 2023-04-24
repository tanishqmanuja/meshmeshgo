package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"

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

func parseNodeId(id string) (int64, error) {
	if len(id) < 3 {
		return 0, errors.New("invalid id string")
	}
	return strconv.ParseInt(id, 0, 32)
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

	for i, graph := range xmlgraphml.Graphs {
		if i == 0 {
			gpath.Graph = simple.NewWeightedDirectedGraph(0, math.Inf(1))
			for _, node := range graph.Nodes {
				node_id, err := parseNodeId(node.Id)
				if err != nil {
					log.Println("ReadGraphXml", err)
					continue
				}
				gpath.Graph.AddNode(simple.NewSimpleNode(int(node_id)))
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

	fmt.Printf("Readed graph with %d nodes and %d edges", gpath.Graph.Nodes().Len(), gpath.Graph.Edges().Len())
}

func (g *GraphPath) GetPath(to int64) ([]graph.Node, error) {
	allShortest := path.DijkstraAllPaths(g.Graph)
	allBetween, weight := allShortest.AllBetween(g.SourceNode, to)
	if len(allBetween) == 0 {
		return nil, fmt.Errorf("no path found between 0x%06X and 0x%06X", g.SourceNode, to)
	}
	log.Printf("Get path from %06X to %06X of lenght %d and weight %f", g.SourceNode, to, len(allBetween[0]), weight)
	return allBetween[0], nil
}

func NewGraphPath(filename string, sourcenode int64) (*GraphPath, error) {
	graph := GraphPath{SourceNode: sourcenode}
	graph.readGraphXml()
	return &graph, nil
}
