package utils

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

func FmtNodeId(nodeid int64) string {
	return fmt.Sprintf("0x%06X", nodeid)
}

func FmtNodeIdHass(nodeid int64) string {
	return fmt.Sprintf("127.%d.%d.%d", (nodeid>>16)&0xFF, (nodeid>>8)&0xFF, nodeid&0xFF)
}

func ForceDebug(force bool, data interface{}) {
	/*var level logrus.Level = logrus.DebugLevel
	if force {
		level = logrus.InfoLevel
	}
	logrus.(level, data)*/
}

func ForceDebugEntry(entry *logrus.Entry, force bool, data interface{}) {
	var level logrus.Level = logrus.DebugLevel
	if force {
		level = logrus.InfoLevel
	}
	entry.Log(level, data)
}

func EncodeToHexEllipsis(data []byte, maxlen int) string {
	str := hex.EncodeToString(data[0:min(len(data), maxlen)])
	if len(data) > maxlen {
		str += "..."
	}
	return str
}

func FindFirstZeroChar(s []byte) int {
	for i, c := range s {
		if c == 0 {
			return i
		}
	}
	return len(s)
}

func TruncateZeros(s []byte) string {
	return string(s[:FindFirstZeroChar(s)])
}

func BackupFile(filename string, backupdir string) {
	if _, err := os.Stat(backupdir); err != nil {
		os.MkdirAll(backupdir, 0755)
	}
	ext := filepath.Ext(filename)
	filenamenoext := strings.TrimSuffix(filename, ext)
	backupfile := filenamenoext + "_" + time.Now().Format("20060102150405") + ext + ".bak"
	if _, err := os.Stat(filename); err == nil {
		os.Rename(filename, filepath.Join(backupdir, backupfile))
	}
}
