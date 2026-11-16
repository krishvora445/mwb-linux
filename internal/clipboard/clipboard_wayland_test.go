//go:build linux

package clipboard

import (
	"testing"
	"time"
)

func TestWaylandClipboardText(t *testing.T) {
	if !isWaylandClipboard() {
		t.Skip("Not in Wayland, skipping Wayland clipboard test")
	}

	m := NewManager(nil, "")
	testStr := "mwb-wayland-test-" + time.Now().Format(time.RFC3339Nano)
	m.setLocalClipboard(testStr)

	got := m.getLocalClipboard()
	if got != testStr {
		t.Fatalf("expected clipboard text %q, got %q", testStr, got)
	}
	t.Logf("Wayland clipboard set/get verified: %q", got)
}
