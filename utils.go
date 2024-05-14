package main

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

func FmtNodeId(nodeid MeshNodeId) string {
	return fmt.Sprintf("0x%06X", nodeid)
}

func FmtNodeIdHass(nodeid MeshNodeId) string {
	return fmt.Sprintf("0.%d.%d.%d", (nodeid>>16)&0xFF, (nodeid>>8)&0xFF, nodeid&0xFF)
}

func ForceDebug(force bool, data interface{}) {
	var level logrus.Level = logrus.DebugLevel
	if force {
		level = logrus.InfoLevel
	}
	log.Log(level, data)
}

func ForceDebugEntry(entry *logrus.Entry, force bool, data interface{}) {
	var level logrus.Level = logrus.DebugLevel
	if force {
		level = logrus.InfoLevel
	}
	entry.Log(level, data)
}
