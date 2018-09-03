package filesys

import (
	"testing"
)

func TestStat(t *testing.T) {
	info, err := Stat("filesys.go")

	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}
	if len(info.FsID) != 16 {
		t.Errorf("Expected FsID to be 16 chars")
	}

	_, err = Stat("x")

	if err == nil {
		t.Errorf("Expected error for unknown file")
	}
}
