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
	"gonum.org/v1/gonum/graph/simple"
)

type XmlGraphml struct {
	XMLName xml.Name   `xml:"graphml"`
	Xmlns   string     `xml:"xmlns"`
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

func intAttribteOfNode(data []XmlData, key string) (int64, error) {
	ret, err := strconv.ParseInt(attribteOfNode(data, key), 0, 32)
	return ret, err
}

func (g *GraphPath) readGraphXml() {
	xmlFile, err := os.Open("meshmesh.graphml")
	if err != nil {
		log.Error("Error opening graph file")
	}

	defer xmlFile.Close()
	byteValue, _ := io.ReadAll(xmlFile)
	var xmlgraphml XmlGraphml
	xml.Unmarshal(byteValue, &xmlgraphml)
	inUseKey := keyForAttribute(xmlgraphml.Keys, "inuse")
	if inUseKey == "" {
		log.Error("Missing inuse field definition in graph")
		return
	}
	isDirectKey := keyForAttribute(xmlgraphml.Keys, "direct")
	if isDirectKey == "" {
		log.Error("Missing direct field definition in graph")
		return
	}
	tagKey := keyForAttribute(xmlgraphml.Keys, "tag")
	if tagKey == "" {
		log.Error("Missing tag field definition in graph")
		return
	}

	weightKey := keyForAttribute(xmlgraphml.Keys, "weight")
	if weightKey == "" {
		log.Error("Missing weight field definition in graph")
		return
	}

	weight2Key := keyForAttribute(xmlgraphml.Keys, "weight2")
	if weight2Key == "" {
		log.Error("Missing weight2 field definition in graph")
		return
	}

	for i, graph := range xmlgraphml.Graphs {
		if i == 0 {
			g.Graph = simple.NewWeightedDirectedGraph(0, math.Inf(1))
			for _, node := range graph.Nodes {
				node_id, err := parseNodeId(node.Id)
				if err != nil {
					log.WithError(err).Error("Error parsing node ID")
					continue
				}

				tag := attribteOfNode(node.Data, tagKey)

				inUse, err := boolAttribteOfNode(node.Data, inUseKey)
				if err != nil {
					log.WithField("node", node.Id).Error("Mssing inuse field in node")
				}

				directPort, err := intAttribteOfNode(node.Data, isDirectKey)
				if err != nil {
					directPort = -1
					isDirectEnabled, err := boolAttribteOfNode(node.Data, isDirectKey)
					if err != nil {
						log.WithField("node", node.Id).Warning("Is direct is implicit assumed false")
					} else {
						if isDirectEnabled {
							directPort = 0
						}
					}
				}

				g.AddNode(node_id)
				g.SetNodeIsInUse(node_id, inUse)
				g.SetNodeDirectPort(node_id, int16(directPort))
				g.SetNodeTag(node_id, tag)
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

				edge := g.Graph.NewWeightedEdge(g.Graph.Node(src_id), g.Graph.Node(dst_id), weight)
				g.Graph.SetWeightedEdge(edge)

			}
		}

	}

	log.WithFields(logrus.Fields{"nodes": g.Graph.Nodes().Len(), "edges": g.Graph.Edges().Len()}).Info("Readed graphml from file")
}

func (g *GraphPath) writeGraphXml(filename string) error {
	var graphml XmlGraphml = XmlGraphml{Xmlns: "http://graphml.graphdrawing.org/xmlns"}
	graphml.Keys = append(graphml.Keys, XmlKey{Id: "d6", For: "edge", AttrName: "weight2", AttrType: "double"})
	graphml.Keys = append(graphml.Keys, XmlKey{Id: "d5", For: "edge", AttrName: "weight", AttrType: "double"})
	graphml.Keys = append(graphml.Keys, XmlKey{Id: "d5", For: "node", AttrName: "direct", AttrType: "int"})
	graphml.Keys = append(graphml.Keys, XmlKey{Id: "d4", For: "node", AttrName: "firmware", AttrType: "string"})
	graphml.Keys = append(graphml.Keys, XmlKey{Id: "d3", For: "node", AttrName: "buggy", AttrType: "bool"})
	graphml.Keys = append(graphml.Keys, XmlKey{Id: "d2", For: "node", AttrName: "discover", AttrType: "bool"})
	graphml.Keys = append(graphml.Keys, XmlKey{Id: "d1", For: "node", AttrName: "inuse", AttrType: "bool"})
	graphml.Keys = append(graphml.Keys, XmlKey{Id: "d0", For: "node", AttrName: "tag", AttrType: "string"})

	graphml.Graphs = append(graphml.Graphs, XmlGraph{})

	var xmlgraph *XmlGraph = &graphml.Graphs[0]
	xmlgraph.EdgeDefault = "undirected"

	nodes := g.Graph.Nodes()
	for nodes.Next() {
		node := nodes.Node()

		_node := XmlNode{Id: FmtNodeId(MeshNodeId(node.ID())), Data: []XmlData{
			{Key: "d0", Text: g.NodeTag(node.ID())},
			{Key: "d1", Text: fmt.Sprintf("%v", g.NodeIsInUse(node.ID()))},
			{Key: "d2", Text: fmt.Sprintf("%v", g.NodeIsDiscovered(node.ID()))},
			{Key: "d5", Text: fmt.Sprintf("%d", g.NodeDirectPort(node.ID()))},
		}}
		xmlgraph.Nodes = append(xmlgraph.Nodes, _node)
	}

	edges := g.Graph.WeightedEdges()
	for edges.Next() {
		edge := edges.WeightedEdge()
		xmlgraph.Edges = append(xmlgraph.Edges, XmlEdge{
			Source: FmtNodeId(MeshNodeId(edge.From().ID())),
			Target: FmtNodeId(MeshNodeId(edge.To().ID())),
			Data: []XmlData{
				{Key: "d5", Text: fmt.Sprintf("%1.2f", edge.Weight())},
			},
		})

	}

	data, err := xml.MarshalIndent(graphml, "", "  ")
	if err != nil {
		return err
	}

	xmlFile, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer xmlFile.Close()
	xmlFile.Write(data)
	return nil
}
