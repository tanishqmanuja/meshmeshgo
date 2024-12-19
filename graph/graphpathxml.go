package graph

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"strconv"

	"github.com/sirupsen/logrus"
	"gonum.org/v1/gonum/graph/simple"
	"leguru.net/m/v2/graphml"
	"leguru.net/m/v2/utils"
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

func ParseNodeIdForGrpah(id string) (int64, error) {
	return parseNodeId(id)

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

/*func intAttribteOfNode(data []XmlData, key string) (int64, error) {
	ret, err := strconv.ParseInt(attribteOfNode(data, key), 0, 32)
	return ret, err
}*/

func (g *GraphPath) readGraph() error {
	xmlFile, err := os.Open("meshmesh.graphml")
	if err != nil {
		return err
	}

	gml := graphml.NewGraphML("meshmesh network")
	err = gml.Decode(xmlFile)
	if err != nil {
		return err
	}

	for i, gr := range gml.Graphs {
		if i == 0 {
			g.Graph = simple.NewWeightedDirectedGraph(0, math.Inf(1))

			fmt.Println(gr.Description)
			for _, n := range gr.Nodes {
				attrs, err := n.GetAttributes()
				if err != nil {
					return err
				}

				id, err := strconv.ParseInt(n.ID, 0, 32)
				if err != nil {
					return err
				}

				inuse, err := strconv.ParseBool(attrs["inuse"].(string))
				if err != nil {
					return err
				}

				g.AddNode(id)
				g.SetNodeIsInUse(id, inuse)
				g.SetNodeTag(id, attrs["tag"].(string))
			}

			for _, e := range gr.Edges {
				attrs, err := e.GetAttributes()
				if err != nil {
					return err
				}

				src, err := strconv.ParseInt(e.Source, 0, 32)
				if err != nil {
					return err
				}
				dst, err := strconv.ParseInt(e.Target, 0, 32)
				if err != nil {
					return err
				}

				weight := attrs["weight"].(float64)

				edge := g.Graph.NewWeightedEdge(g.Graph.Node(src), g.Graph.Node(dst), weight)
				g.Graph.SetWeightedEdge(edge)
			}
		}
	}
	return nil
}

func (g *GraphPath) readGraphXml() {
	xmlFile, err := os.Open("meshmesh.graphml")
	if err != nil {
		logrus.Error("Error opening graph file")
	}

	defer xmlFile.Close()
	byteValue, _ := io.ReadAll(xmlFile)
	var xmlgraphml XmlGraphml
	xml.Unmarshal(byteValue, &xmlgraphml)
	inUseKey := keyForAttribute(xmlgraphml.Keys, "inuse")
	if inUseKey == "" {
		logrus.Error("Missing inuse field definition in graph")
		return
	}
	tagKey := keyForAttribute(xmlgraphml.Keys, "tag")
	if tagKey == "" {
		logrus.Error("Missing tag field definition in graph")
		return
	}
	weightKey := keyForAttribute(xmlgraphml.Keys, "weight")
	if weightKey == "" {
		logrus.Error("Missing weight field definition in graph")
		return
	}
	weight2Key := keyForAttribute(xmlgraphml.Keys, "weight2")
	if weight2Key == "" {
		logrus.Error("Missing weight2 field definition in graph")
		return
	}

	for i, graph := range xmlgraphml.Graphs {
		if i == 0 {
			g.Graph = simple.NewWeightedDirectedGraph(0, math.Inf(1))
			for _, node := range graph.Nodes {
				node_id, err := parseNodeId(node.Id)
				if err != nil {
					logrus.WithError(err).Error("Error parsing node ID")
					continue
				}

				tag := attribteOfNode(node.Data, tagKey)

				inUse, err := boolAttribteOfNode(node.Data, inUseKey)
				if err != nil {
					logrus.WithField("node", node.Id).Error("Mssing inuse field in node")
				}

				g.AddNode(node_id)
				g.SetNodeIsInUse(node_id, inUse)
				g.SetNodeTag(node_id, tag)
			}

			for _, edge := range graph.Edges {
				src_id, err := parseNodeId(edge.Source)
				if err != nil {
					logrus.Println("ReadGraphXml", err)
					continue
				}
				dst_id, err := parseNodeId(edge.Target)
				if err != nil {
					logrus.Println("ReadGraphXml", err)
					continue
				}
				weight, err := strconv.ParseFloat(edge.Data[0].Text, 32)
				if err != nil {
					logrus.Println("ReadGraphXml", err)
					continue
				}

				edge := g.Graph.NewWeightedEdge(g.Graph.Node(src_id), g.Graph.Node(dst_id), weight)
				g.Graph.SetWeightedEdge(edge)

			}
		}

	}

	logrus.WithFields(logrus.Fields{"nodes": g.Graph.Nodes().Len(), "edges": g.Graph.Edges().Len()}).Info("Readed graphml from file")
}

func (g *GraphPath) WriteGraph(filename string) error {
	gml := graphml.NewGraphML("meshmesh network")

	gml.RegisterKey(graphml.KeyForNode, "tag", "tag of node", reflect.String, "")
	gml.RegisterKey(graphml.KeyForNode, "inuse", "is node in use", reflect.Bool, true)
	gml.RegisterKey(graphml.KeyForNode, "discover", "state variable for dicvery", reflect.Bool, false)
	gml.RegisterKey(graphml.KeyForNode, "buggy", "state variable fr functional status", reflect.Bool, false)
	gml.RegisterKey(graphml.KeyForNode, "firmware", "the node firmware revision", reflect.String, "")
	gml.RegisterKey(graphml.KeyForEdge, "weight", "the node firmware revision", reflect.Float32, 0.0)
	gml.RegisterKey(graphml.KeyForEdge, "weight2", "the node firmware revision", reflect.Float32, 0.0)

	gr, err := gml.AddGraph("the graph", graphml.EdgeDirectionUndirected, map[string]interface{}{})
	if err != nil {
		return err
	}

	nodes := g.Graph.Nodes()
	for nodes.Next() {
		node := nodes.Node()

		attributes := map[string]interface{}{
			"inuse":      g.NodeIsInUse(node.ID()),
			"discovered": g.NodeIsDiscovered(node.ID()),
		}

		gr.AddNode(attributes, utils.FmtNodeId(node.ID()), "the input node")
	}

	edges := g.Graph.WeightedEdges()
	for edges.Next() {
		edge := edges.WeightedEdge()

		n1 := gr.GetNode(utils.FmtNodeId(edge.From().ID()))
		n2 := gr.GetNode(utils.FmtNodeId(edge.To().ID()))

		attributes := map[string]interface{}{
			"weight": edge.Weight(),
		}

		gr.AddEdge(n1, n2, attributes, graphml.EdgeDirectionDefault, "the first level")
	}

	xmlFile, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer xmlFile.Close()
	err = gml.Encode(xmlFile, true)
	return err
}

func (g *GraphPath) WriteGraphXml(filename string) error {
	var graphml XmlGraphml = XmlGraphml{Xmlns: "http://graphml.graphdrawing.org/xmlns"}
	graphml.Keys = append(graphml.Keys, XmlKey{Id: "d6", For: "edge", AttrName: "weight2", AttrType: "double"})
	graphml.Keys = append(graphml.Keys, XmlKey{Id: "d5", For: "edge", AttrName: "weight", AttrType: "double"})
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

		_node := XmlNode{Id: utils.FmtNodeId(node.ID()), Data: []XmlData{
			{Key: "d0", Text: g.NodeTag(node.ID())},
			{Key: "d1", Text: fmt.Sprintf("%v", g.NodeIsInUse(node.ID()))},
			{Key: "d2", Text: fmt.Sprintf("%v", g.NodeIsDiscovered(node.ID()))},
		}}
		xmlgraph.Nodes = append(xmlgraph.Nodes, _node)
	}

	edges := g.Graph.WeightedEdges()
	for edges.Next() {
		edge := edges.WeightedEdge()
		xmlgraph.Edges = append(xmlgraph.Edges, XmlEdge{
			Source: utils.FmtNodeId(edge.From().ID()),
			Target: utils.FmtNodeId(edge.To().ID()),
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
