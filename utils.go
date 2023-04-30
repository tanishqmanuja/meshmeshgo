package main

import "fmt"

func FmtNodeId(nodeid MeshNodeId) string {
	return fmt.Sprintf("0x%06X", nodeid)
}
