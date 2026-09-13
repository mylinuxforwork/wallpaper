package tui

import (
	"strings"
	"testing"

	"github.com/nerdexecutive/ne-image-sorter/internal/domain"
)

func TestMenuNavigationStaysInBounds(t *testing.T) {
	m := newMenuModel(readyConfig())
	for i := 0; i < 20; i++ {
		m, _ = m.Update(key("down"))
	}
	if m.cursor != len(menuItems)-1 {
		t.Errorf("cursor = %d after over-scrolling, want %d", m.cursor, len(menuItems)-1)
	}
	for i := 0; i < 20; i++ {
		m, _ = m.Update(key("up"))
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d after over-scrolling up, want 0", m.cursor)
	}
}

func TestMenuWarnsWhenNotConfigured(t *testing.T) {
	unset := newMenuModel(domain.DefaultConfig()).View()
	if !strings.Contains(unset, "Not ready") {
		t.Error("menu must warn when the directories are unset")
	}
	if !strings.Contains(unset, "(not set)") {
		t.Error("menu must render unset paths as a visible placeholder")
	}
	if ready := newMenuModel(readyConfig()).View(); strings.Contains(ready, "Not ready") {
		t.Error("menu must not warn when the config is valid")
	}
}

func TestMenuQuitKey(t *testing.T) {
	if _, cmd := newMenuModel(readyConfig()).Update(key("q")); cmd == nil {
		t.Error("q produced no command, want quit")
	}
}

func TestMenuEnterOpensTheSelectedScreen(t *testing.T) {
	m := newMenuModel(readyConfig())
	if _, cmd := m.Update(key("enter")); cmd == nil {
		t.Fatal("enter produced no command")
	} else if msg, ok := cmd().(switchMsg); !ok || msg.screen != ScreenSort {
		t.Errorf("enter on the first item = %v, want a switch to the sort screen", cmd())
	}
}

func TestFoldersEditingAndDefaultDestination(t *testing.T) {
	m := newFoldersModel(domain.DefaultConfig())
	for _, r := range "/tmp/pics" {
		m, _ = m.Update(key(string(r)))
	}
	if m.values[0] != "/tmp/pics" {
		t.Fatalf("source = %q, want /tmp/pics", m.values[0])
	}

	m, _ = m.Update(key("backspace"))
	if m.values[0] != "/tmp/pic" {
		t.Errorf("backspace left %q", m.values[0])
	}

	m, _ = m.Update(key("ctrl+d"))
	if !strings.HasPrefix(m.values[1], "/tmp/pic/") {
		t.Errorf("ctrl+d destination = %q, want it under the source", m.values[1])
	}

	m, _ = m.Update(key("ctrl+u"))
	if m.values[0] != "" {
		t.Errorf("ctrl+u left %q, want an empty field", m.values[0])
	}
}

func TestFoldersTabSwitchesField(t *testing.T) {
	m := newFoldersModel(domain.DefaultConfig())
	if m.field != 0 {
		t.Fatal("folders must open on the source field")
	}
	m, _ = m.Update(key("tab"))
	if m.field != 1 {
		t.Errorf("field = %d after tab, want 1", m.field)
	}
	m, _ = m.Update(key("tab"))
	if m.field != 0 {
		t.Errorf("field = %d after a second tab, want 0", m.field)
	}
}

func TestFoldersRejectsAMissingSource(t *testing.T) {
	m := newFoldersModel(domain.DefaultConfig())
	m.values = [2]string{"/definitely/not/here", "/definitely/not/here/sub"}
	m, cmd := m.Update(key("enter"))
	if cmd != nil {
		t.Error("enter must not save when the source does not exist")
	}
	if !strings.Contains(m.err, "does not exist") {
		t.Errorf("err = %q, want it to name the missing directory", m.err)
	}
}

func TestFoldersRejectsIdenticalDirectories(t *testing.T) {
	m := newFoldersModel(domain.DefaultConfig())
	m.values = [2]string{"/tmp", "/tmp"}
	m, cmd := m.Update(key("enter"))
	if cmd != nil || m.err == "" {
		t.Error("enter must refuse a destination identical to the source")
	}
}

func TestFoldersSavesAValidPair(t *testing.T) {
	dir := t.TempDir()
	m := newFoldersModel(domain.DefaultConfig())
	m.values = [2]string{dir, dir + "/rejects"}
	m, cmd := m.Update(key("enter"))
	if cmd == nil {
		t.Fatalf("enter on a valid pair produced no command, err = %q", m.err)
	}
	if m.cfg.SourceDir != dir {
		t.Errorf("saved source = %q, want %q", m.cfg.SourceDir, dir)
	}
}

func TestExpandHome(t *testing.T) {
	if got := expandHome("/absolute/path"); got != "/absolute/path" {
		t.Errorf("expandHome left an absolute path as %q", got)
	}
	if got := expandHome("~/pics"); strings.HasPrefix(got, "~") {
		t.Errorf("expandHome(~/pics) = %q, want the tilde resolved", got)
	}
	if got := expandHome("./relative"); got != "./relative" {
		t.Errorf("expandHome altered a relative path to %q", got)
	}
}
