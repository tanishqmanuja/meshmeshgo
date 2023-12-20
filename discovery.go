package main

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/sirupsen/logrus"
	"gonum.org/v1/gonum/graph"
)

type _weightsStats struct {
	From float64
	To   float64
}

func _rssi2weight(rssi int16) float64 {
	if rssi <= 0 {
		rssi *= -2
	} else if rssi > 44 {
		rssi = 44
	}
	return 1.0 - float64(rssi)/45.0
}

func _mapOfNeighbors(g graph.WeightedDirected, n graph.Node, w map[int64]_weightsStats) error {
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

func _findNextNode(g *GraphPath) MeshNode {
	nodes := g.Graph.Nodes()
	var found_node MeshNode
	var found_weight float64 = 1e9
	for nodes.Next() {
		_node := nodes.Node()
		node := _node.(MeshNode)
		if node.inUse && !node.Discovered {
			path, weight, err := g.GetPath(node.id)
			if weight < found_weight {
				found_weight = weight
				found_node = node
			}
			log.Println("path", path, weight, err)
			//node.Discovered
		}
	}
	return found_node
}

func DoDiscovery(serial *SerialConnection) error {
	var err error
	var reply1 interface{}

	var oldWeights map[int64]_weightsStats = make(map[int64]_weightsStats)
	var newWeights map[int64]_weightsStats = make(map[int64]_weightsStats)

	g, err := NewGraphPath(int64(serial.LocalNode))
	if err != nil {
		return err
	}

	currentNode := _findNextNode(g)
	for ; currentNode.GetInUse(); currentNode = _findNextNode(g) {
		protocol := directProtocol
		if currentNode.ID() != g.SourceNode {
			protocol = unicastProtocol
		}

		logrus.Printf("Start dicover of node %06X", currentNode.ID())
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
			g.ChangeEdgetWeight(currentNode.ID(), int64(tableItem.NodeId), _rssi2weight(tableItem.Rssi2), _rssi2weight(tableItem.Rssi1))
		}

		logrus.Printf("Map of neighbors of node %06X is completed", currentNode.ID())
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
		logrus.Printf("|----|--------|----------|----------|-------|")
		logrus.Printf("| N. | ID     | Prev.    | Curr.    | Delta |")
		logrus.Printf("|----|--------|--------.-|----------|-------|")
		for id, w := range newWeights {
			i += 1
			w1 := oldWeights[id]
			pre := fmt.Sprintf("%1.2f,%1.2f", w1.To, w1.From)
			post := fmt.Sprintf("%1.2f,%1.2f", w.To, w.From)
			logrus.Printf("| %02X | %06X | %s | %s | _____ |", i, id, pre, post)
		}
		logrus.Printf("|----|--------|----------|----------|-------|")
	}

	return nil
}
