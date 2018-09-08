package filesys

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStat(t *testing.T) {
	info, err := Stat("filesys.go")

	assert.Nil(t, err, "Unexpected error")
	assert.Equal(t, 16, len(info.fsID), "Expected FsID to be 16 chars")
	assert.NotEqual(t, 0, info.Size())

	_, err = Stat("x")

	if err == nil {
		t.Errorf("Expected error for unknown file")
	}
}
