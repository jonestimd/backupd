package database

import (
	"path/filepath"
)

// Internal struct for building file paths.
type pathNode struct {
	names  []string // path node names in bottom-up order (child, parent, ...)
	nextID *string  // node ID of one of the path's parents
}

func newPathNode(names []string, parentID *string) *pathNode {
	return &pathNode{names, parentID}
}

func (path *pathNode) String() string {
	reorder := make([]string, len(path.names))
	for i, j := 0, len(path.names)-1; j >= 0; i, j = i+1, j-1 {
		reorder[j] = path.names[i]
	}
	return string(filepath.Separator) + filepath.Join(reorder...)
}

func (path *pathNode) append(name string, nextID *string) *pathNode {
	return newPathNode(append(path.names, name), nextID)
}
