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

func (g *GraphPath) readGraphXml() {
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
			g.Graph = simple.NewWeightedDirectedGraph(0, math.Inf(1))
			for _, node := range graph.Nodes {
				node_id, err := parseNodeId(node.Id)
				if err != nil {
					log.Println("ReadGraphXml", err)
					continue
				}
				inUse, err := boolAttribteOfNode(node.Data, inusekey)
				if err != nil {
					log.WithField("node", node.Id).Error("Mssing inuse field in node")
				}
				g.AddNode(node_id)
				g.SetNodeIsInUse(node_id, inUse)
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
