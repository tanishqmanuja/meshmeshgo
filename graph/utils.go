package graph

import (
	"fmt"

	"leguru.net/m/v2/logger"
	"leguru.net/m/v2/utils"
)

func FmtNodePath(gpath *GraphPath, nodeid int64) string {
	var _path string
	path, _, err := gpath.GetPath(nodeid)
	if err == nil {
		for _, p := range path {
			if len(_path) > 0 {
				_path += " > "
			}
			_path += utils.FmtNodeId(p)
		}
	}
	return _path
}

func PrintTable(graph *GraphPath) {
	if !graph.NodeExists(graph.SourceNode) {
		logger.WithField("node", graph.SourceNode).Fatal("Local node does not exists in grpah")
	}

	fmt.Println("Coordinator node is " + utils.FmtNodeId(graph.SourceNode))

	fmt.Println("|----------|----------------|--------------------|------|--------------------------------------------------|------|")
	fmt.Println("| Node Id  | Node Address   | Node Tag           | Port | Path                                             | Wei. |")
	fmt.Println("|----------|----------------|--------------------|------|--------------------------------------------------|------|")

	inuse := graph.GetAllInUse()
	for _, d := range inuse {
		nid := d

		var _path string
		path, weight, err := graph.GetPath(d)
		if err == nil {
			for _, p := range path {
				if len(_path) > 0 {
					_path += " > "
				}
				_path += utils.FmtNodeId(p)
			}
		}

		fmt.Printf("| %s | %15s | %-18s | %4d | %-48s | %3.2f |\n", utils.FmtNodeId(nid), utils.FmtNodeIdHass(nid), graph.NodeTag(d), 6053, _path, weight)
	}

}
