//go:build linux

package capture

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func getHyprlandSocketPath() (string, bool) {
	sig := os.Getenv("HYPRLAND_INSTANCE_SIGNATURE")
	if sig == "" {
		return "", false
	}
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	sockPath := filepath.Join(runtimeDir, "hypr", sig, ".socket.sock")
	if _, err := os.Stat(sockPath); err == nil {
		return sockPath, true
	}
	return "", false
}

func isHyprland() bool {
	_, ok := getHyprlandSocketPath()
	return ok
}

type hyprlandPointer struct {
	sockPath string
}

func newHyprlandPointer() (*hyprlandPointer, error) {
	path, ok := getHyprlandSocketPath()
	if !ok {
		return nil, fmt.Errorf("hyprland socket not found")
	}
	p := &hyprlandPointer{sockPath: path}
	if _, _, err := p.Position(); err != nil {
		return nil, fmt.Errorf("hyprland pointer probe failed: %w", err)
	}
	return p, nil
}

func queryHyprlandSocket(sockPath, cmd string) (string, error) {
	conn, err := net.DialTimeout("unix", sockPath, 200*time.Millisecond)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(500 * time.Millisecond))

	if _, err := conn.Write([]byte(cmd)); err != nil {
		return "", err
	}
	out, err := io.ReadAll(conn)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (p *hyprlandPointer) querySocket(cmd string) (string, error) {
	return queryHyprlandSocket(p.sockPath, cmd)
}

func (p *hyprlandPointer) Position() (int32, int32, error) {
	out, err := p.querySocket("cursorpos")
	if err != nil {
		return -1, -1, err
	}
	parts := strings.Split(strings.TrimSpace(out), ",")
	if len(parts) != 2 {
		return -1, -1, fmt.Errorf("unexpected cursorpos format: %q", out)
	}
	x, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 32)
	if err != nil {
		return -1, -1, fmt.Errorf("parse cursor x: %w", err)
	}
	y, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 32)
	if err != nil {
		return -1, -1, fmt.Errorf("parse cursor y: %w", err)
	}
	return int32(x), int32(y), nil
}

func (p *hyprlandPointer) MoveTo(x, y int32) error {
	// First try modern Hyprland Lua dispatcher (Hyprland >= 0.56)
	cmd := fmt.Sprintf("eval return hl.dispatch(hl.dsp.cursor.move({ x = %d, y = %d }))", x, y)
	res, err := p.querySocket(cmd)
	if err == nil && strings.TrimSpace(res) == "ok" {
		return nil
	}

	// Fallback to older dispatch syntax
	fallbackCmd := fmt.Sprintf("dispatch movecursor %d %d", x, y)
	resFallback, errFallback := p.querySocket(fallbackCmd)
	if errFallback == nil && strings.TrimSpace(resFallback) == "ok" {
		return nil
	}

	if err != nil {
		return err
	}
	return fmt.Errorf("movecursor failed: %s (fallback: %v, %s)", res, errFallback, resFallback)
}

func (p *hyprlandPointer) Close() {
	// Sockets are closed per call
}

type hyprlandMonitorJSON struct {
	ID       int32   `json:"id"`
	Name     string  `json:"name"`
	Width    int32   `json:"width"`
	Height   int32   `json:"height"`
	Scale    float64 `json:"scale"`
	X        int32   `json:"x"`
	Y        int32   `json:"y"`
	Focused  bool    `json:"focused"`
	Disabled bool    `json:"disabled"`
}

// GetScreenSizeHyprland queries monitor geometry from Hyprland IPC.
func GetScreenSizeHyprland() (ScreenInfo, error) {
	path, ok := getHyprlandSocketPath()
	if !ok {
		return ScreenInfo{}, fmt.Errorf("hyprland socket not found")
	}
	out, err := queryHyprlandSocket(path, "j/monitors")
	if err != nil {
		return ScreenInfo{}, err
	}
	var monitors []hyprlandMonitorJSON
	if err := json.Unmarshal([]byte(out), &monitors); err != nil {
		return ScreenInfo{}, fmt.Errorf("unmarshal monitors: %w", err)
	}
	if len(monitors) == 0 {
		return ScreenInfo{}, fmt.Errorf("no monitors found")
	}

	var maxWidth, maxHeight int32
	var focusedMonitor *hyprlandMonitorJSON

	for i := range monitors {
		m := &monitors[i]
		if m.Disabled {
			continue
		}
		if m.Focused {
			focusedMonitor = m
		}
		right := m.X + m.Width
		bottom := m.Y + m.Height
		if right > maxWidth {
			maxWidth = right
		}
		if bottom > maxHeight {
			maxHeight = bottom
		}
	}

	if len(monitors) == 1 && focusedMonitor != nil {
		return ScreenInfo{Width: focusedMonitor.Width, Height: focusedMonitor.Height}, nil
	}
	if maxWidth > 0 && maxHeight > 0 {
		return ScreenInfo{Width: maxWidth, Height: maxHeight}, nil
	}
	return ScreenInfo{Width: monitors[0].Width, Height: monitors[0].Height}, nil
}
