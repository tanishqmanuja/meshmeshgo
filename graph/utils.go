package graph

import "leguru.net/m/v2/utils"

func FmtNodePath(gpath *GraphPath, nodeid int64) string {
	var _path string
	path, _, err := gpath.GetPath(nodeid)
	if err == nil {
		for _, p := range path {
			if len(_path) > 0 {
				_path += " > "
			}
			_path += utils.FmtNodeId(uint32(p))
		}
	}
	return _path
}
