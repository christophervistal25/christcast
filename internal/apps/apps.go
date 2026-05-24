// Package apps discovers installed XDG desktop applications by walking
// standard .desktop directories, parses their Name + display rules, and
// returns store.FileInfo entries that the index can mix with file results.
package apps

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/chriscast/chriscast/internal/store"
)

// Dirs returns the ordered list of directories that may contain .desktop files
// on this system, honoring XDG_DATA_HOME and XDG_DATA_DIRS.
func Dirs() []string {
	var dirs []string

	// system dirs (XDG_DATA_DIRS, default to /usr/local/share:/usr/share)
	systemRaw := os.Getenv("XDG_DATA_DIRS")
	if systemRaw == "" {
		systemRaw = "/usr/local/share:/usr/share"
	}
	for _, d := range strings.Split(systemRaw, ":") {
		if d == "" {
			continue
		}
		dirs = append(dirs, filepath.Join(d, "applications"))
	}

	// user dir (XDG_DATA_HOME, default to ~/.local/share)
	userBase := os.Getenv("XDG_DATA_HOME")
	if userBase == "" {
		if home, err := os.UserHomeDir(); err == nil {
			userBase = filepath.Join(home, ".local", "share")
		}
	}
	if userBase != "" {
		dirs = append(dirs, filepath.Join(userBase, "applications"))
	}

	// flatpak + snap exports (best-effort)
	dirs = append(dirs,
		"/var/lib/flatpak/exports/share/applications",
		"/var/lib/snapd/desktop/applications",
	)
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs,
			filepath.Join(home, ".local/share/flatpak/exports/share/applications"),
		)
	}
	return dirs
}

// Scan returns FileInfo entries for every visible .desktop application
// in standard dirs. Later .desktop files with the same basename override
// earlier ones (matching XDG search-order semantics — user overrides system).
func Scan() []store.FileInfo {
	type hit struct {
		path string
		name string
		icon string
		mod  int64
	}
	byID := map[string]hit{} // keyed by .desktop basename

	for _, dir := range Dirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if !strings.HasSuffix(name, ".desktop") {
				continue
			}
			full := filepath.Join(dir, name)
			display, icon, ok := parse(full)
			if !ok {
				continue
			}
			info, _ := e.Info()
			var mod int64
			if info != nil {
				mod = info.ModTime().Unix()
			}
			byID[name] = hit{path: full, name: display, icon: icon, mod: mod}
		}
	}

	out := make([]store.FileInfo, 0, len(byID))
	for _, h := range byID {
		out = append(out, store.FileInfo{
			Path:    h.path,
			Base:    h.name,
			ModTime: h.mod,
			IsApp:   true,
			Icon:    h.icon,
		})
	}
	return out
}

// parse returns the display Name and Icon of the .desktop file plus a bool
// indicating whether the entry should be shown to the user (Type=Application,
// not NoDisplay/Hidden, no OnlyShowIn/NotShowIn exclusion). The Icon value is
// returned raw — it may be an absolute path or an icon-theme name; resolution
// is left to the renderer.
func parse(path string) (string, string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)

	inEntry := false
	var name, icon string
	isApp := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			inEntry = line == "[Desktop Entry]"
			continue
		}
		if !inEntry {
			continue
		}
		key, val, ok := splitKV(line)
		if !ok {
			continue
		}
		switch key {
		case "Name":
			if name == "" {
				name = val
			}
		case "Icon":
			if icon == "" {
				icon = val
			}
		case "Type":
			if strings.EqualFold(val, "Application") {
				isApp = true
			}
		case "NoDisplay", "Hidden":
			if isTrue(val) {
				return "", "", false
			}
		}
	}
	if !isApp || name == "" {
		return "", "", false
	}
	return name, icon, true
}

// splitKV splits a "Key=Value" line, returning false for lines without
// an `=` (locale-suffixed Name[fr]= is rejected — caller handles locale
// rules separately). Whitespace around the `=` is tolerated.
func splitKV(line string) (key, value string, ok bool) {
	eq := strings.IndexByte(line, '=')
	if eq <= 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:eq])
	value = strings.TrimSpace(line[eq+1:])
	// reject locale-suffixed keys like "Name[fr]"
	if strings.IndexByte(key, '[') >= 0 {
		return "", "", false
	}
	return key, value, true
}

// isTrue returns true for any spec-legal boolean truthy value.
func isTrue(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "yes":
		return true
	}
	return false
}
