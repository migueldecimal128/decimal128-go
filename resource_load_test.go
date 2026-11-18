package decimal128

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResourceLoadLoadDqAdd(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(resourcesDir(), "dectest", "dqAdd.decTest"))
	if err != nil {
		t.Fatalf("dqAdd.decTest not found: %v", err)
	}
	if len(data) == 0 || !strings.Contains(string(data), "dqadd") {
		t.Error("dqAdd.decTest content")
	}
}
