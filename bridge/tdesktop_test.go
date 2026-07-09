package bridge

import (
	"strings"
	"testing"
)

func TestTDesktopEmptyDir(t *testing.T) {
	dir := t.TempDir()
	if _, err := FromTDesktop(dir, nil); err == nil {
		t.Fatal("expected error for empty tdata dir")
	} else if !strings.Contains(err.Error(), "read tdata") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTDesktopMissingDir(t *testing.T) {
	if _, err := FromTDesktopAll("/no/such/tdata", nil); err == nil {
		t.Fatal("expected error for missing tdata dir")
	}
}

func TestConvertRoutesDirToTDesktop(t *testing.T) {
	dir := t.TempDir()
	_, err := Convert(dir)
	if err == nil {
		t.Fatal("expected error routing dir to tdesktop")
	}
	if !strings.Contains(err.Error(), "key_data") {
		t.Errorf("expected tdesktop error, got: %v", err)
	}
}
