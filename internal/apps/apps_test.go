package apps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeDesktop(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseVisibleApp(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "firefox.desktop")
	writeDesktop(t, p, `[Desktop Entry]
Type=Application
Name=Firefox
Exec=firefox %u
Icon=firefox
`)
	name, icon, ok := parse(p)
	if !ok || name != "Firefox" {
		t.Fatalf("want Firefox/ok, got %q/%v", name, ok)
	}
	if icon != "firefox" {
		t.Fatalf("want Icon=firefox, got %q", icon)
	}
}

func TestParseCapturesAbsoluteIcon(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "thing.desktop")
	writeDesktop(t, p, `[Desktop Entry]
Type=Application
Name=Thing
Exec=thing
Icon=/usr/share/pixmaps/thing.png
`)
	_, icon, ok := parse(p)
	if !ok || icon != "/usr/share/pixmaps/thing.png" {
		t.Fatalf("want absolute icon path, got %q/%v", icon, ok)
	}
}

func TestScanPopulatesIcon(t *testing.T) {
	dir := t.TempDir()
	writeDesktop(t, filepath.Join(dir, "applications", "iconapp.desktop"), `[Desktop Entry]
Type=Application
Name=IconApp
Exec=iconapp
Icon=iconapp
`)
	t.Setenv("XDG_DATA_DIRS", dir)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	for _, h := range Scan() {
		if h.Base == "IconApp" {
			if h.Icon != "iconapp" {
				t.Fatalf("want Icon=iconapp, got %q", h.Icon)
			}
			return
		}
	}
	t.Fatal("IconApp not found")
}

func TestParseSkipsNoDisplay(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "hidden.desktop")
	writeDesktop(t, p, `[Desktop Entry]
Type=Application
Name=ShouldBeHidden
Exec=foo
NoDisplay=true
`)
	if _, _, ok := parse(p); ok {
		t.Fatal("NoDisplay=true must be filtered")
	}
}

func TestParseSkipsHidden(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "h.desktop")
	writeDesktop(t, p, `[Desktop Entry]
Type=Application
Name=Nope
Hidden=true
Exec=foo
`)
	if _, _, ok := parse(p); ok {
		t.Fatal("Hidden=true must be filtered")
	}
}

func TestParseSkipsNonApplication(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "link.desktop")
	writeDesktop(t, p, `[Desktop Entry]
Type=Link
Name=Link
URL=https://example.com
`)
	if _, _, ok := parse(p); ok {
		t.Fatal("Type=Link must be filtered (Type=Application only)")
	}
}

func TestParseIgnoresOtherSections(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "x.desktop")
	writeDesktop(t, p, `[Desktop Action New]
Name=NewWindow
Exec=foo --new

[Desktop Entry]
Type=Application
Name=RealName
Exec=foo
`)
	name, _, ok := parse(p)
	if !ok || name != "RealName" {
		t.Fatalf("want RealName/ok, got %q/%v", name, ok)
	}
}

func TestParseEmptyOrMissingName(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "noname.desktop")
	writeDesktop(t, p, `[Desktop Entry]
Type=Application
Exec=foo
`)
	if _, _, ok := parse(p); ok {
		t.Fatal("missing Name must be filtered")
	}
}

func TestScanDeduplicatesByBasename(t *testing.T) {
	// system dir holds an entry; user dir overrides it.
	sys := t.TempDir()
	user := t.TempDir()

	writeDesktop(t, filepath.Join(sys, "applications", "thing.desktop"), `[Desktop Entry]
Type=Application
Name=SystemName
Exec=thing
`)
	writeDesktop(t, filepath.Join(user, "applications", "thing.desktop"), `[Desktop Entry]
Type=Application
Name=UserName
Exec=thing
`)

	t.Setenv("XDG_DATA_DIRS", sys)
	t.Setenv("XDG_DATA_HOME", user)
	t.Setenv("HOME", t.TempDir()) // isolate flatpak/snap probes

	hits := Scan()
	var seen *string
	for i := range hits {
		if hits[i].Path == filepath.Join(user, "applications", "thing.desktop") {
			seen = &hits[i].Base
		}
		if hits[i].Path == filepath.Join(sys, "applications", "thing.desktop") {
			t.Errorf("system entry should be overridden by user entry, got both")
		}
	}
	if seen == nil || *seen != "UserName" {
		t.Fatalf("user override missing or wrong name, got hits=%+v", hits)
	}
}

func TestScanReturnsIsAppMarker(t *testing.T) {
	dir := t.TempDir()
	writeDesktop(t, filepath.Join(dir, "applications", "marker.desktop"), `[Desktop Entry]
Type=Application
Name=Marker
Exec=marker
`)
	t.Setenv("XDG_DATA_DIRS", dir)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	for _, h := range Scan() {
		if h.Base == "Marker" {
			if !h.IsApp {
				t.Fatal("Scan must mark entries IsApp=true")
			}
			return
		}
	}
	t.Fatal("Marker not found")
}

func TestParsePrefersPlainNameOverLocale(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "fox.desktop")
	writeDesktop(t, p, `[Desktop Entry]
Type=Application
Name[fr]=Renard
Name=Firefox
Exec=firefox
`)
	name, _, ok := parse(p)
	if !ok || name != "Firefox" {
		t.Fatalf("want Firefox (plain Name=), got %q/%v", name, ok)
	}
}

func TestParseHandlesCommentsAndIndentedLines(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "comments.desktop")
	writeDesktop(t, p, `# leading comment
[Desktop Entry]
# inner comment
   Type=Application
	Name=Indented
    Exec=foo
`)
	name, _, ok := parse(p)
	if !ok || name != "Indented" {
		t.Fatalf("want Indented/ok despite comments + indentation, got %q/%v", name, ok)
	}
}

func TestParseFirstNameWins(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "order.desktop")
	// Plain Name= appears after locale variants; locales must not be picked,
	// and the FIRST plain Name= encountered must win over any later Name=.
	writeDesktop(t, p, `[Desktop Entry]
Type=Application
Name[de]=Feuerfuchs
Name[es]=Zorro
Name=FirstWins
Name=ShouldBeIgnored
Exec=foo
`)
	name, _, ok := parse(p)
	if !ok || name != "FirstWins" {
		t.Fatalf("want FirstWins, got %q/%v", name, ok)
	}
}

func TestParseToleratesWhitespaceAndCase(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "ws.desktop")
	writeDesktop(t, p, `[Desktop Entry]
Type = Application
Name = Whitespaced
NoDisplay = False
Hidden = NO
Exec=foo
`)
	name, _, ok := parse(p)
	if !ok || name != "Whitespaced" {
		t.Fatalf("want Whitespaced, got %q/%v", name, ok)
	}
}

func TestParseNoDisplayCapitalized(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "cap.desktop")
	writeDesktop(t, p, `[Desktop Entry]
Type=Application
Name=ShouldBeSkipped
NoDisplay=True
Exec=foo
`)
	if _, _, ok := parse(p); ok {
		t.Fatal("NoDisplay=True (capitalized) must filter the entry")
	}
}

func TestParseIconReturned(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "icon.desktop")
	writeDesktop(t, p, `[Desktop Entry]
Type=Application
Name=Iconed
Icon=firefox
Exec=foo
`)
	_, icon, ok := parse(p)
	if !ok || icon != "firefox" {
		t.Fatalf("want icon=firefox, got %q/%v", icon, ok)
	}
}

func TestScanMissingDirsDoesNotPanic(t *testing.T) {
	xdgMissing := filepath.Join(t.TempDir(), "does-not-exist")
	homeMissing := filepath.Join(t.TempDir(), "also-missing")
	fakeHome := t.TempDir()
	t.Setenv("XDG_DATA_DIRS", xdgMissing)
	t.Setenv("XDG_DATA_HOME", homeMissing)
	t.Setenv("HOME", fakeHome)

	hits := Scan() // must not panic
	// Scan() also probes hard-coded /var/lib/flatpak and /var/lib/snapd paths
	// that aren't controllable via env, so we can't assert len==0 on a real
	// host. The contract we verify is: no hit comes from the missing dirs we
	// pointed XDG at.
	for _, h := range hits {
		if strings.HasPrefix(h.Path, xdgMissing) ||
			strings.HasPrefix(h.Path, homeMissing) ||
			strings.HasPrefix(h.Path, fakeHome) {
			t.Errorf("hit from missing/isolated dir leaked: %+v", h)
		}
	}
}

func TestDirsFallbackWhenXDGDataDirsEmpty(t *testing.T) {
	t.Setenv("XDG_DATA_DIRS", "")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	dirs := Dirs()
	wantLocal := filepath.Join("/usr/local/share", "applications")
	wantShare := filepath.Join("/usr/share", "applications")
	var sawLocal, sawShare bool
	for _, d := range dirs {
		if d == wantLocal {
			sawLocal = true
		}
		if d == wantShare {
			sawShare = true
		}
	}
	if !sawLocal || !sawShare {
		t.Fatalf("Dirs() must include %q and %q when XDG_DATA_DIRS empty, got %v", wantLocal, wantShare, dirs)
	}
}
