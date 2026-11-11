package database

import (
	"path/filepath"
	"strings"
	"testing"

	"jsfraz/whisper-server/utils"

	"github.com/google/uuid"
)

func TestMediaFilePath(t *testing.T) {
	utils.GetSingleton().Config.MediaDir = t.TempDir()
	id := uuid.NewString()

	path, err := MediaFilePath(id)
	if err != nil {
		t.Fatalf("valid id rejected: %v", err)
	}
	if filepath.Base(path) != id {
		t.Fatalf("path %q does not end with the media id", path)
	}
	if !strings.HasPrefix(path, utils.GetSingleton().Config.MediaDir) {
		t.Fatalf("path %q escapes the media directory", path)
	}

	for _, invalid := range []string{"", "..", "../etc/passwd", "not-a-uuid", "abc/def", "/tmp/file"} {
		if _, err := MediaFilePath(invalid); err == nil {
			t.Errorf("invalid id %q was accepted", invalid)
		}
	}
}
