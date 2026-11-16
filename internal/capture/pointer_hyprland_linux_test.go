//go:build linux

package capture

import (
	"testing"
)

func TestHyprlandPointer(t *testing.T) {
	if !isHyprland() {
		t.Skip("Hyprland not running, skipping live Hyprland test")
	}

	p, err := newHyprlandPointer()
	if err != nil {
		t.Fatalf("newHyprlandPointer failed: %v", err)
	}
	defer p.Close()

	x, y, err := p.Position()
	if err != nil {
		t.Fatalf("Position() failed: %v", err)
	}
	t.Logf("Hyprland cursor position: %d, %d", x, y)
	if x < 0 || y < 0 {
		t.Errorf("Unexpected negative cursor coordinates: (%d, %d)", x, y)
	}

	screen, err := GetScreenSizeHyprland()
	if err != nil {
		t.Fatalf("GetScreenSizeHyprland() failed: %v", err)
	}
	t.Logf("Hyprland screen size: %dx%d", screen.Width, screen.Height)
	if screen.Width <= 0 || screen.Height <= 0 {
		t.Errorf("Unexpected invalid screen size: %dx%d", screen.Width, screen.Height)
	}
}
