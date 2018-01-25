package database

import (
	"path/filepath"
)

type pathNode struct {
	names  []string
	nextId *string
}

func newPathNode(names []string, parentId *string) *pathNode {
	return &pathNode{names, parentId}
}

func (path *pathNode) String() string {
	reorder := make([]string, len(path.names))
	for i, j := 0, len(path.names)-1; j >= 0; i, j = i+1, j-1 {
		reorder[j] = path.names[i]
	}
	return string(filepath.Separator) + filepath.Join(reorder...)
}

func (path *pathNode) append(name string, nextId *string) *pathNode {
	return newPathNode(append(path.names, name), nextId)
}
