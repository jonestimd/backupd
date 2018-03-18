package database

import (
	"reflect"
	"testing"
)

func TestBoltTx_InsertFile(t *testing.T) {
	remoteId := "remoteId"
	mimeType := "text/plain"
	localId := "localId"
	md5checksum := "md5checksum"
	modifiedDate := "2018-01-01"
	b := makeFileBucket()
	tx := boltTx{b, nil}
	file := RemoteFile{"name", mimeType, 16, &md5checksum, []string{"parentId"}, &modifiedDate, &localId, &remoteId}

	tx.InsertFile(remoteId, file.Name, mimeType, file.Size, file.Md5Checksum, file.ParentIds, *file.LastModified, &localId)

	if !reflect.DeepEqual(toBytes(&file), b.keyValues[remoteId]) {
		t.Errorf("Expected remoteFile %v to equal %v", toRemoteFile(b.keyValues[remoteId]), file)
	}
}

func checkPath(t *testing.T, expected string, actual string) {
	if expected != actual {
		t.Errorf("Expected file id '%s' to equal '%s'", actual, expected)
	}
}

func TestBoltTx_SetPaths(t *testing.T) {
	parent := RemoteFile{"parent", "text/plain", 16, nil, []string{"rootId"}, nil, nil, nil}
	file := RemoteFile{"name", "text/plain", 16, nil, []string{"parent"}, nil, nil, nil}
	fileBucket := makeFileBucket(&file, &parent)
	pathBucket := makeMockBucket()
	tx := boltTx{fileBucket, pathBucket}

	tx.SetPaths()

	if len(pathBucket.keyValues) != 2 {
		t.Error("Expected paths for each file")
	}
	checkPath(t, "parent", string(pathBucket.keyValues["/parent"]))
	checkPath(t, "name", string(pathBucket.keyValues["/parent/name"]))
}

func checkPathCallback(t *testing.T, path string, expected string, pathMap map[string]string) {
	if expected != pathMap[path] {
		t.Errorf("Expected callback to be called with '%s', '%s'", path, expected)
	}
}

func TestBoltTx_ForEachPath(t *testing.T) {
	pathBucket := makeMockBucket()
	pathBucket.keyValues["/parent"] = []byte("parent")
	pathBucket.keyValues["/parent/name"] = []byte("name")
	tx := boltTx{nil, pathBucket}
	pathMap := make(map[string]string)

	err := tx.ForEachPath(func(path string, id string) error {
		pathMap[path] = id
		return nil
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(pathMap) != 2 {
		t.Error("Expected callback to be called twice")
	}
	checkPathCallback(t, "/parent", "parent", pathMap)
	checkPathCallback(t, "/parent/name", "name", pathMap)
}

func TestGetPath(t *testing.T) {
	tests := []struct {
		description string
		bucket      *mockBucket
		expected    []string
	}{
		{"unknown file", makeFileBucket(&RemoteFile{Name: "unknown"}), []string{}},
		{"nil parents", makeFileBucket(&RemoteFile{Name: "file1"}), []string{"/file1"}},
		{"empty parents", makeFileBucket(&RemoteFile{Name: "file1", ParentIds: []string{}}), []string{"/file1"}},
		{"unknown parent", makeFileBucket(&RemoteFile{Name: "file1", ParentIds: []string{"parent"}}), []string{"/file1"}},
		{"one parent", makeFileBucket(
			&RemoteFile{Name: "file1", ParentIds: []string{"parent"}},
			&RemoteFile{Name: "parent", ParentIds: []string{"gp"}}), []string{"/parent/file1"}},
		{"parent with empty parents", makeFileBucket(
			&RemoteFile{Name: "file1", ParentIds: []string{"parent"}},
			&RemoteFile{Name: "parent", ParentIds: []string{}}), []string{"/parent/file1"}},
		{"two parents", makeFileBucket(
			&RemoteFile{Name: "file1", ParentIds: []string{"parent1", "parent2"}},
			&RemoteFile{Name: "parent1", ParentIds: []string{"gp"}},
			&RemoteFile{Name: "parent2", ParentIds: []string{"gp"}}), []string{"/parent1/file1", "/parent2/file1"}},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actual := getPaths(test.bucket, "file1")
			if !reflect.DeepEqual(actual, test.expected) {
				t.Errorf("Expected paths %v to equal %v", actual, test.expected)
			}
		})
	}
}

func TestGetFile(t *testing.T) {
	b := makeFileBucket(&RemoteFile{Name: "existing", Size: 123})
	tests := []struct {
		description string
		fileId      string
		expected    *RemoteFile
	}{
		{"existing file", "existing", toRemoteFile(b.keyValues["existing"])},
		{"non-existing file", "unknown", nil},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actual := getFile(b, &test.fileId)
			if !reflect.DeepEqual(actual, test.expected) {
				t.Errorf("Expected remoteFile %v to equal %v", actual, test.expected)
			}
		})
	}
}
