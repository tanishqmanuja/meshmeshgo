package meshmesh

import (
	"errors"
	"fmt"
	"math"
	"time"

	log "github.com/sirupsen/logrus"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"

	gra "leguru.net/m/v2/graph"
	"leguru.net/m/v2/utils"
)

type DiscoveryProcedure struct {
	serial      *SerialConnection
	graph       *gra.GraphPath
	currentNode gra.MeshNode
	Neighbors   map[int64]discWeights
}

func (d *DiscoveryProcedure) CurrentNode() int64 {
	return d.currentNode.ID()
}

type _weightsStats struct {
	From float64
	To   float64
}

type discWeights struct {
	Current float64
	Next    float64
}

func _rssi2weight(rssi int16) float64 {
	if rssi <= 0 {
		rssi *= -2
	} else if rssi > 44 {
		rssi = 44
	}
	return 1.0 - float64(rssi)/45.0
}

func _mapOfNeighbors(g *simple.WeightedDirectedGraph, n graph.Node, w map[int64]_weightsStats) error {
	neighbors := g.From(n.ID())
	for neighbors.Next() {
		neighbor := neighbors.Node()
		weightTo, ok := g.Weight(n.ID(), neighbor.ID())
		if !ok {
			return errors.New("corrupted graph")
		}
		weightFrom, ok := g.Weight(neighbor.ID(), n.ID())
		if !ok {
			return errors.New("corrupted graph")
		}
		w[neighbor.ID()] = _weightsStats{To: weightTo, From: weightFrom}
	}
	return nil
}

func neighborsFromGraph(g *simple.WeightedDirectedGraph, n graph.Node, w map[int64]discWeights) error {
	neighbors := g.From(n.ID())
	for neighbors.Next() {
		neighbor := neighbors.Node()
		weightTo, ok := g.Weight(n.ID(), neighbor.ID())
		if !ok {
			return errors.New("corrupted graph")
		}
		weightFrom, ok := g.Weight(neighbor.ID(), n.ID())
		if !ok {
			return errors.New("corrupted graph")
		}
		w[neighbor.ID()] = discWeights{Current: 1.0, Next: math.Min(weightTo, weightFrom)}
	}
	return nil
}

func neighborsAdavance(w map[int64]discWeights) {
	for i, d := range w {
		w[i] = discWeights{Current: d.Next, Next: 1.0}
	}
}

func neighborsToGraph(g *gra.GraphPath, w map[int64]discWeights) {

}

func _updateNeighbor(w map[int64]discWeights, id int64, rssi1, rssi2 float64) error {
	if _, exists := w[id]; exists {
		w[id] = discWeights{Current: w[id].Current, Next: math.Min(rssi1, rssi2)}
	} else {
		w[id] = discWeights{Current: 1.0, Next: math.Min(rssi1, rssi2)}
	}
	return nil
}

func _findNextNode(g *gra.GraphPath) gra.MeshNode {
	nodes := g.Graph.Nodes()
	var found_node gra.MeshNode
	var found_weight float64 = 1e9
	for nodes.Next() {
		_node := nodes.Node()
		node := _node.(gra.MeshNode)
		if g.NodeIsInUse(node.ID()) && !g.NodeIsDiscovered(node.ID()) {
			path, weight, err := g.GetPath(node.ID())
			if weight < found_weight {
				found_weight = weight
				found_node = node
			}
			log.Println("path", path, weight, err)
			g.SetNodeIsDiscovered(node.ID(), true)
		}
	}
	return found_node
}

func DoDiscovery(serial *SerialConnection) error {
	var err error
	var reply1 interface{}

	var oldWeights map[int64]_weightsStats = make(map[int64]_weightsStats)
	var newWeights map[int64]_weightsStats = make(map[int64]_weightsStats)

	g, err := gra.NewGraphPath(int64(serial.LocalNode))
	if err != nil {
		return err
	}

	currentNode := _findNextNode(g)
	for ; g.NodeIsDiscovered(currentNode.ID()); currentNode = _findNextNode(g) {
		protocol := directProtocol
		if currentNode.ID() != g.SourceNode {
			protocol = UnicastProtocol
		}

		log.Printf("Start dicover of node %06X", currentNode.ID())
		_mapOfNeighbors(g.Graph, currentNode, oldWeights)

		_, err = serial.SendReceiveApiProt(DiscResetTableApiRequest{}, protocol, MeshNodeId(currentNode.ID()))
		if err != nil {
			return err
		}

		_, err = serial.SendReceiveApiProt(DiscStartDiscoverApiRequest{Mask: 0, Filter: 0, Slotnum: 100}, protocol, MeshNodeId(currentNode.ID()))
		if err != nil {
			return err
		}

		time.Sleep(3 * time.Second)

		reply1, err = serial.SendReceiveApiProt(DiscTableSizeApiRequest{}, protocol, MeshNodeId(currentNode.ID()))
		if err != nil {
			return err
		}
		tableSize, ok := reply1.(DiscTableSizeApiReply)
		if !ok {
			return errors.New("comunication error")
		}
		log.Printf("Table size is %d", tableSize.Size)
		for i := uint8(0); i < tableSize.Size; i++ {

			reply1, err = serial.SendReceiveApiProt(DiscTableItemGetApiRequest{}, protocol, MeshNodeId(currentNode.ID()))
			if err != nil {
				return err
			}
			tableItem, ok := reply1.(DiscTableItemGetApiReply)
			if !ok {
				return errors.New("comunication error")
			}

			log.Printf("Query of row %d rssi1 %d rssi2 %d", i+1, tableItem.Rssi1, tableItem.Rssi2)
			g.ChangeEdgeWeight(currentNode.ID(), int64(tableItem.NodeId), _rssi2weight(tableItem.Rssi2), _rssi2weight(tableItem.Rssi1))
		}

		log.Printf("Map of neighbors of node %06X is completed", currentNode.ID())
		_mapOfNeighbors(g.Graph, currentNode, newWeights)

		for id := range newWeights {
			if _, exists := oldWeights[id]; exists {

			} else {
				oldWeights[id] = _weightsStats{From: math.NaN(), To: math.NaN()}
			}
		}

		for id := range oldWeights {
			if _, exists := newWeights[id]; exists {

			} else {
				newWeights[id] = _weightsStats{From: math.NaN(), To: math.NaN()}
			}
		}

		var i int = 0
		log.Printf("|----|--------|-----------|-----------|-------|")
		log.Printf("| N. | ID     | Prev.     | Curr.     | Delta |")
		log.Printf("|----|--------|-----------|-----------|-------|")
		for id, w := range newWeights {
			i += 1
			w1 := oldWeights[id]
			pre := fmt.Sprintf("%1.2f,%1.2f", w1.To, w1.From)
			post := fmt.Sprintf("%1.2f,%1.2f", w.To, w.From)
			log.Printf("| %02X | %06X | %s | %s | _____ |", i, id, pre, post)
		}
		log.Printf("|----|--------|-----------|-----------|-------|")
	}

	g.WriteGraphXml("discovery.graphml")
	return nil
}

func (d *DiscoveryProcedure) Init() error {
	var err error

	d.graph, err = gra.NewGraphPath(int64(d.serial.LocalNode))
	if err != nil {
		return err
	}

	d.currentNode = _findNextNode(d.graph)
	d.Neighbors = make(map[int64]discWeights)
	neighborsFromGraph(d.graph.Graph, d.currentNode, d.Neighbors)
	return nil
}

func (d *DiscoveryProcedure) Step() error {
	protocol := directProtocol
	if d.currentNode.ID() != d.graph.SourceNode {
		protocol = UnicastProtocol
	}

	log.Printf("Start dicover of node %06X", d.currentNode.ID())

	_, err := d.serial.SendReceiveApiProt(DiscResetTableApiRequest{}, protocol, MeshNodeId(d.currentNode.ID()))
	if err != nil {
		return err
	}

	_, err = d.serial.SendReceiveApiProt(DiscStartDiscoverApiRequest{Mask: 0, Filter: 0, Slotnum: 100}, protocol, MeshNodeId(d.currentNode.ID()))
	if err != nil {
		return err
	}

	time.Sleep(3 * time.Second)

	reply1, err := d.serial.SendReceiveApiProt(DiscTableSizeApiRequest{}, protocol, MeshNodeId(d.currentNode.ID()))
	if err != nil {
		return err
	}
	tableSize, ok := reply1.(DiscTableSizeApiReply)
	if !ok {
		return errors.New("comunication error")
	}

	neighborsAdavance(d.Neighbors)
	for i := uint8(0); i < tableSize.Size; i++ {

		reply1, err = d.serial.SendReceiveApiProt(DiscTableItemGetApiRequest{Index: i}, protocol, MeshNodeId(d.currentNode.ID()))
		if err != nil {
			return err
		}
		tableItem, ok := reply1.(DiscTableItemGetApiReply)
		if !ok {
			return errors.New("comunication error")
		}

		log.Printf("Query of row %d node %s rssi1 %d rssi2 %d", i, utils.FmtNodeId(int64(tableItem.NodeId)), tableItem.Rssi1, tableItem.Rssi2)
		_updateNeighbor(d.Neighbors, int64(tableItem.NodeId), _rssi2weight(tableItem.Rssi1), _rssi2weight(tableItem.Rssi2))
		//d.graph.ChangeEdgeWeight(d.currentNode.ID(), int64(tableItem.NodeId), _rssi2weight(tableItem.Rssi2), _rssi2weight(tableItem.Rssi1))
	}
	return err
}

func (d *DiscoveryProcedure) Save() error {
	if d.currentNode.ID() == -1 {
		return errors.New("discovery is inactive")
	}
	neighborsToGraph(d.graph, d.Neighbors)
	return nil

}

func NewDiscoveryProcedure(serial *SerialConnection) *DiscoveryProcedure {
	return &DiscoveryProcedure{serial: serial, currentNode: gra.NewMeshNode(-1)}
}
