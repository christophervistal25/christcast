//go:build gtk

package ui

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chriscast/chriscast/internal/config"
	"github.com/chriscast/chriscast/internal/index"
	"github.com/chriscast/chriscast/internal/search"
	"github.com/chriscast/chriscast/internal/store"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

const (
	winWidth   = 720
	winHeight  = 480
	winCompact = 64 // compact = entry only, no results panel
	debounceMS = 50
)

type App struct {
	cfg       *config.Config
	ix        *index.Index
	win       *gtk.Window
	entry     *gtk.Entry
	list      *gtk.ListBox
	scrolled  *gtk.ScrolledWindow
	sepTop    *gtk.Separator
	hintSep   *gtk.Separator
	hintBar   *gtk.Box
	results   []search.Result
	pending   glib.SourceHandle
	daemon    bool
	expanded  bool
}

func Run(cfg *config.Config, ix *index.Index) error {
	a, err := NewApp(cfg, ix)
	if err != nil {
		return err
	}
	a.Show()
	a.Main()
	return nil
}

func NewApp(cfg *config.Config, ix *index.Index) (*App, error) {
	gtk.Init(nil)
	a := &App{cfg: cfg, ix: ix}
	if err := a.build(); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *App) SetDaemon(v bool) { a.daemon = v }

func (a *App) Main() { gtk.Main() }

func (a *App) Show() { a.ShowAt(0) }

// ShowAt opens the overlay; timestamp should be the X11 event time that
// triggered the show (from the hotkey handler) so window managers honor
// the focus request and don't apply focus-steal prevention.
func (a *App) ShowAt(timestamp uint32) {
	// always reset on Show — Raycast-like compact open.
	a.entry.SetText("")
	a.results = nil
	a.clearList()
	a.expanded = true // force setExpanded to apply
	a.setExpanded(false)
	// mark heavy widgets "don't auto-show" so ShowAll won't re-show them.
	a.sepTop.SetNoShowAll(true)
	a.scrolled.SetNoShowAll(true)
	a.hintSep.SetNoShowAll(true)
	a.hintBar.SetNoShowAll(true)
	a.win.ShowAll()
	a.win.Resize(winWidth, winCompact)
	a.positionTop()
	if timestamp != 0 {
		a.win.PresentWithTime(timestamp)
	} else {
		a.win.Present()
	}
	a.entry.GrabFocus()
}

// positionTop centers the window horizontally on the active monitor and
// anchors it near the top (Raycast-style — not vertically centered).
func (a *App) positionTop() {
	const topOffset = 120
	display, err := gdk.DisplayGetDefault()
	if err != nil || display == nil {
		return
	}
	// prefer the monitor where the cursor is; fall back to monitor 0.
	monitor, merr := display.GetPrimaryMonitor()
	if merr != nil || monitor == nil {
		monitor, _ = display.GetMonitor(0)
	}
	if monitor == nil {
		return
	}
	geom := monitor.GetWorkarea()
	if geom == nil {
		geom = monitor.GetGeometry()
	}
	if geom == nil {
		return
	}
	x := geom.GetX() + (geom.GetWidth()-winWidth)/2
	y := geom.GetY() + topOffset
	a.win.Move(x, y)
}

func (a *App) Hide() {
	if a.pending != 0 {
		glib.SourceRemove(a.pending)
		a.pending = 0
	}
	a.win.Hide()
}

func (a *App) build() error {
	w, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		return err
	}
	a.win = w
	w.SetTitle("chriscast")
	w.SetDefaultSize(winWidth, winCompact)
	w.SetDecorated(false)
	w.SetResizable(false)
	w.SetKeepAbove(true)
	w.SetSkipTaskbarHint(true)
	w.SetSkipPagerHint(true)
	w.SetPosition(gtk.WIN_POS_NONE)
	w.SetTypeHint(gdk.WINDOW_TYPE_HINT_DIALOG)
	addClass(&w.Widget, "cct-overlay")

	// enable per-pixel transparency so the rounded corners show the desktop
	// behind them instead of a black square.
	if screen := w.GetScreen(); screen != nil {
		if visual, _ := screen.GetRGBAVisual(); visual != nil {
			w.SetVisual(visual)
		}
	}
	w.SetAppPaintable(true)

	if err := loadCSS(); err != nil {
		return err
	}

	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return err
	}
	w.Add(box)

	a.entry, err = gtk.EntryNew()
	if err != nil {
		return err
	}
	a.entry.SetPlaceholderText("Search files…")
	a.entry.SetHasFrame(false)
	addClass(&a.entry.Widget, "cct-entry")
	box.PackStart(a.entry, false, false, 0)

	sep, err := gtk.SeparatorNew(gtk.ORIENTATION_HORIZONTAL)
	if err != nil {
		return err
	}
	addClass(&sep.Widget, "cct-sep")
	box.PackStart(sep, false, false, 0)
	a.sepTop = sep

	scrolled, err := gtk.ScrolledWindowNew(nil, nil)
	if err != nil {
		return err
	}
	scrolled.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	scrolled.SetVExpand(true)
	box.PackStart(scrolled, true, true, 0)
	a.scrolled = scrolled

	a.list, err = gtk.ListBoxNew()
	if err != nil {
		return err
	}
	a.list.SetSelectionMode(gtk.SELECTION_SINGLE)
	a.list.SetActivateOnSingleClick(false)
	addClass(&a.list.Widget, "cct-list")
	scrolled.Add(a.list)

	// Bottom hint bar — persistent Raycast-style key legend.
	hintSep, err := gtk.SeparatorNew(gtk.ORIENTATION_HORIZONTAL)
	if err != nil {
		return err
	}
	addClass(&hintSep.Widget, "cct-sep")
	box.PackStart(hintSep, false, false, 0)
	a.hintSep = hintSep

	hintBar, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return err
	}
	addClass(&hintBar.Widget, "cct-hint-bar")
	a.hintBar = hintBar

	hintInner, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return err
	}
	hintInner.SetHAlign(gtk.ALIGN_END)
	for _, h := range []string{
		"↵  open",
		"Ctrl+↵  reveal",
		"Ctrl+O  terminal",
		"Ctrl+C  copy path",
		"Esc  close",
	} {
		lbl, err := gtk.LabelNew(h)
		if err != nil {
			return err
		}
		addClass(&lbl.Widget, "cct-hint")
		hintInner.PackStart(lbl, false, false, 0)
	}
	hintBar.PackStart(hintInner, true, true, 0)
	box.PackStart(hintBar, false, false, 0)

	a.entry.Connect("changed", a.onChanged)
	a.entry.Connect("activate", a.onActivate)
	a.list.Connect("row-activated", a.onRowActivated)
	w.Connect("key-press-event", a.onKey)
	w.Connect("destroy", a.quitOrHide)
	w.Connect("focus-out-event", func() bool {
		a.quitOrHide()
		return false
	})

	a.entry.GrabFocus()
	// start compact — only the search field is visible until results arrive.
	a.setExpanded(false)
	return nil
}

// setExpanded toggles between compact (search field only) and full
// (search field + results panel + hint bar). Resizes the window to match.
func (a *App) setExpanded(on bool) {
	if a.expanded == on && a.scrolled != nil {
		return
	}
	a.expanded = on
	if on {
		a.sepTop.SetNoShowAll(false)
		a.scrolled.SetNoShowAll(false)
		a.hintSep.SetNoShowAll(false)
		a.hintBar.SetNoShowAll(false)
		a.sepTop.Show()
		a.scrolled.Show()
		a.hintSep.Show()
		a.hintBar.Show()
		a.win.Resize(winWidth, winHeight)
	} else {
		a.sepTop.Hide()
		a.scrolled.Hide()
		a.hintSep.Hide()
		a.hintBar.Hide()
		a.win.Resize(winWidth, winCompact)
	}
}

func (a *App) quitOrHide() {
	if a.daemon {
		a.Hide()
		return
	}
	gtk.MainQuit()
}

func (a *App) onChanged() {
	if a.pending != 0 {
		glib.SourceRemove(a.pending)
		a.pending = 0
	}
	id := glib.TimeoutAdd(debounceMS, func() bool {
		a.pending = 0
		a.runQuery()
		return false
	})
	a.pending = id
}

func (a *App) runQuery() {
	text, _ := a.entry.GetText()
	if text == "" {
		a.results = nil
		a.clearList()
		a.setExpanded(false)
		return
	}
	a.results = search.Search(a.ix, text, a.cfg.MaxResults)
	a.renderList()
}

func (a *App) clearList() {
	// collect first, then remove — mutating during Foreach is undefined.
	var widgets []*gtk.Widget
	a.list.GetChildren().Foreach(func(item any) {
		if w, ok := item.(*gtk.Widget); ok {
			widgets = append(widgets, w)
		}
	})
	for _, w := range widgets {
		a.list.Remove(w)
	}
}

func (a *App) showEmpty(msg string) {
	a.clearList()
	lbl, err := gtk.LabelNew(msg)
	if err != nil {
		return
	}
	addClass(&lbl.Widget, "cct-empty")
	lbl.SetHAlign(gtk.ALIGN_CENTER)
	a.list.Add(lbl)
	a.list.ShowAll()
}

func (a *App) renderList() {
	a.clearList()
	if len(a.results) == 0 {
		a.setExpanded(true)
		a.showEmpty("No matches.")
		return
	}
	a.setExpanded(true)
	for _, r := range a.results {
		row := buildRow(r)
		a.list.Add(row)
	}
	a.list.ShowAll()
	if first := a.list.GetRowAtIndex(0); first != nil {
		a.list.SelectRow(first)
	}
}

func buildRow(r search.Result) *gtk.ListBoxRow {
	row, _ := gtk.ListBoxRowNew()
	hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	if iconW := appIconWidget(r.File); iconW != nil {
		hbox.PackStart(iconW, false, false, 0)
	} else {
		icon, _ := gtk.LabelNew(glyphFor(r.File))
		icon.SetHAlign(gtk.ALIGN_CENTER)
		icon.SetSizeRequest(28, -1)
		addClass(&icon.Widget, "cct-icon")
		hbox.PackStart(icon, false, false, 0)
	}

	vbox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	label := r.File.Base
	if r.File.IsDir {
		label += "/"
	}
	base, _ := gtk.LabelNew(label)
	base.SetHAlign(gtk.ALIGN_START)
	addClass(&base.Widget, "cct-base")
	path, _ := gtk.LabelNew(filepath.Dir(r.File.Path))
	path.SetHAlign(gtk.ALIGN_START)
	path.SetEllipsize(3) // PANGO_ELLIPSIZE_END
	addClass(&path.Widget, "cct-path")
	vbox.PackStart(base, false, false, 0)
	vbox.PackStart(path, false, false, 0)
	hbox.PackStart(vbox, true, true, 0)

	row.Add(hbox)
	// stash result via row index — retrieved on activate
	return row
}

// appIconWidget returns a 28-wide *gtk.Image rendering the .desktop file's
// Icon= value (absolute path → load from file, otherwise → themed icon name).
// Returns nil when fi is not an app, has no icon, or icon resolution fails —
// caller should fall back to the Unicode glyph label.
func appIconWidget(fi *store.FileInfo) gtk.IWidget {
	if fi == nil || !fi.IsApp || fi.Icon == "" {
		return nil
	}
	const px = 24
	var img *gtk.Image
	if filepath.IsAbs(fi.Icon) {
		if _, err := os.Stat(fi.Icon); err != nil {
			return nil
		}
		pix, err := gdk.PixbufNewFromFileAtScale(fi.Icon, px, px, true)
		if err != nil || pix == nil {
			return nil
		}
		im, err := gtk.ImageNewFromPixbuf(pix)
		if err != nil || im == nil {
			return nil
		}
		img = im
	} else {
		im, err := gtk.ImageNewFromIconName(fi.Icon, gtk.ICON_SIZE_DND)
		if err != nil || im == nil {
			return nil
		}
		im.SetPixelSize(px)
		img = im
	}
	img.SetHAlign(gtk.ALIGN_CENTER)
	img.SetSizeRequest(28, -1)
	addClass(&img.Widget, "cct-icon")
	return img
}

// glyphFor returns a small Unicode glyph chosen by file kind / extension.
func glyphFor(fi *store.FileInfo) string {
	if fi.IsApp {
		return "✦"
	}
	if fi.IsDir {
		return "📁"
	}
	ext := strings.ToLower(filepath.Ext(fi.Path))
	switch ext {
	case ".go", ".py", ".js", ".ts", ".tsx", ".rb", ".rs", ".java",
		".c", ".cpp", ".h", ".hpp", ".php":
		return "⟨⟩"
	case ".md", ".txt", ".rst":
		return "≡"
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg":
		return "▣"
	case ".pdf":
		return "▤"
	case ".zip", ".tar", ".gz", ".bz2", ".xz", ".7z", ".rar":
		return "▢"
	case ".mp3", ".wav", ".flac", ".mp4", ".mkv", ".mov", ".webm":
		return "▶"
	}
	return "·"
}

func (a *App) onActivate() {
	row := a.list.GetSelectedRow()
	if row == nil {
		return
	}
	a.openIndex(row.GetIndex())
}

func (a *App) onRowActivated(_ *gtk.ListBox, row *gtk.ListBoxRow) {
	if row == nil {
		return
	}
	a.openIndex(row.GetIndex())
}

func (a *App) openIndex(i int) {
	if i < 0 || i >= len(a.results) {
		return
	}
	fi := a.results[i].File
	if fi.IsApp {
		launchApp(fi.Path)
	} else {
		open(fi.Path)
	}
	a.quitOrHide()
}

func (a *App) revealIndex(i int) {
	if i < 0 || i >= len(a.results) {
		return
	}
	open(filepath.Dir(a.results[i].File.Path))
	a.quitOrHide()
}

func (a *App) terminalIndex(i int) {
	if i < 0 || i >= len(a.results) {
		return
	}
	fi := a.results[i].File
	dir := fi.Path
	if !fi.IsDir {
		dir = filepath.Dir(dir)
	}
	openTerminal(dir)
	a.quitOrHide()
}

func (a *App) copyIndex(i int) {
	if i < 0 || i >= len(a.results) {
		return
	}
	path := a.results[i].File.Path
	clip, err := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
	if err != nil {
		return
	}
	clip.SetText(path)
	clip.Store()
	notify("Copied to clipboard", path)
}

// notify fires a desktop notification via libnotify (notify-send). Silent
// no-op if notify-send isn't installed. Kept minimal: extra flags like
// --app-name / --hint=desktop-entry can cause GNOME to filter the toast
// when the user has muted the matching app entry, which is the wrong
// failure mode here.
func notify(title, body string) {
	bin, err := exec.LookPath("notify-send")
	if err != nil {
		return
	}
	cmd := exec.Command(bin, title, body)
	if err := cmd.Start(); err != nil {
		return
	}
	go cmd.Wait()
}

func (a *App) onKey(_ *gtk.Window, ev *gdk.Event) bool {
	key := gdk.EventKeyNewFromEvent(ev)
	val := key.KeyVal()
	mods := gdk.ModifierType(key.State())
	ctrl := mods&gdk.CONTROL_MASK != 0

	switch val {
	case gdk.KEY_Escape:
		a.quitOrHide()
		return true
	case gdk.KEY_Down:
		a.move(+1)
		return true
	case gdk.KEY_Up:
		a.move(-1)
		return true
	case gdk.KEY_Right:
		// drill into selected directory: replace entry text with its
		// path + "/" so dir-browse mode lists its children.
		if pos := a.entry.GetPosition(); pos == a.entryLen() {
			row := a.list.GetSelectedRow()
			if row != nil {
				idx := row.GetIndex()
				if idx >= 0 && idx < len(a.results) && a.results[idx].File.IsDir {
					path := a.results[idx].File.Path
					a.entry.SetText(path + "/")
					a.entry.SetPosition(-1) // end
					return true
				}
			}
		}
	case gdk.KEY_Return, gdk.KEY_KP_Enter:
		row := a.list.GetSelectedRow()
		if row == nil {
			return true
		}
		idx := row.GetIndex()
		if ctrl {
			a.revealIndex(idx)
		} else {
			a.openIndex(idx)
		}
		return true
	case gdk.KEY_o, gdk.KEY_O:
		if ctrl {
			row := a.list.GetSelectedRow()
			if row != nil {
				a.terminalIndex(row.GetIndex())
				return true
			}
		}
	case gdk.KEY_c, gdk.KEY_C:
		if ctrl {
			start, end, _ := a.entry.GetSelectionBounds()
			if start == end { // no text selected in entry — copy path
				row := a.list.GetSelectedRow()
				if row != nil {
					a.copyIndex(row.GetIndex())
					a.quitOrHide()
					return true
				}
			}
		}
	}
	return false
}

func (a *App) entryLen() int {
	t, _ := a.entry.GetText()
	return len([]rune(t))
}

func (a *App) move(delta int) {
	row := a.list.GetSelectedRow()
	idx := 0
	if row != nil {
		idx = row.GetIndex() + delta
	}
	if idx < 0 {
		idx = 0
	}
	if idx >= len(a.results) {
		idx = len(a.results) - 1
	}
	if target := a.list.GetRowAtIndex(idx); target != nil {
		a.list.SelectRow(target)
		a.scrollTo(target)
	}
}

// scrollTo nudges the ScrolledWindow vadjustment so the row is fully visible.
func (a *App) scrollTo(row *gtk.ListBoxRow) {
	if a.scrolled == nil || row == nil {
		return
	}
	adj := a.scrolled.GetVAdjustment()
	if adj == nil {
		return
	}
	alloc := row.GetAllocation()
	rowTop := float64(alloc.GetY())
	rowH := float64(alloc.GetHeight())
	if rowH <= 0 {
		return
	}
	page := adj.GetPageSize()
	cur := adj.GetValue()
	if rowTop < cur {
		adj.SetValue(rowTop)
	} else if rowTop+rowH > cur+page {
		adj.SetValue(rowTop + rowH - page)
	}
}

func open(p string) {
	cmd := exec.Command("xdg-open", p)
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(stderr(), "xdg-open:", err)
		return
	}
	go cmd.Wait()
}

// launchApp launches a .desktop application file. Prefers `gtk-launch`
// (handles Exec field-codes, env, working dir correctly), then `gio
// launch`, falling back to a manual parse + exec of the Exec line.
func launchApp(desktopPath string) {
	id := strings.TrimSuffix(filepath.Base(desktopPath), ".desktop")
	if bin, err := exec.LookPath("gtk-launch"); err == nil {
		cmd := exec.Command(bin, id)
		if err := cmd.Start(); err == nil {
			go cmd.Wait()
			return
		}
	}
	if bin, err := exec.LookPath("gio"); err == nil {
		cmd := exec.Command(bin, "launch", desktopPath)
		if err := cmd.Start(); err == nil {
			go cmd.Wait()
			return
		}
	}
	// last resort: parse Exec=, strip %-codes, exec via sh -c.
	exe := readExec(desktopPath)
	if exe == "" {
		fmt.Fprintln(stderr(), "launchApp: no usable launcher (need gtk-launch or gio)")
		return
	}
	cmd := exec.Command("sh", "-c", exe+" &")
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(stderr(), "launchApp fallback:", err)
		return
	}
	go cmd.Wait()
}

// readExec extracts the Exec= line from a .desktop file and strips XDG
// field codes (%f, %u, %F, %U, %i, %c, %k) which we don't pass anyway.
func readExec(desktopPath string) string {
	data, err := os.ReadFile(desktopPath)
	if err != nil {
		return ""
	}
	inEntry := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
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
		if strings.HasPrefix(line, "Exec=") {
			raw := strings.TrimPrefix(line, "Exec=")
			out := make([]rune, 0, len(raw))
			skip := false
			for _, r := range raw {
				if skip {
					skip = false
					continue
				}
				if r == '%' {
					skip = true
					continue
				}
				out = append(out, r)
			}
			return strings.TrimSpace(string(out))
		}
	}
	return ""
}

// terminalCandidate is one possible terminal emulator + how to tell it
// which working directory to start in.
type terminalCandidate struct {
	bin  string
	args func(dir string) []string
}

var terminalCandidates = []terminalCandidate{
	// Generic XDG terminal launcher (newest).
	{"xdg-terminal-exec", func(d string) []string { return []string{} }},
	// Common Linux terminals — most accept some form of "start here".
	{"gnome-terminal", func(d string) []string { return []string{"--working-directory=" + d} }},
	{"konsole", func(d string) []string { return []string{"--workdir", d} }},
	{"kitty", func(d string) []string { return []string{"-d", d} }},
	{"alacritty", func(d string) []string { return []string{"--working-directory", d} }},
	{"wezterm", func(d string) []string { return []string{"start", "--cwd", d} }},
	{"foot", func(d string) []string { return []string{"--working-directory=" + d} }},
	{"tilix", func(d string) []string { return []string{"--working-directory=" + d} }},
	{"xfce4-terminal", func(d string) []string { return []string{"--working-directory=" + d} }},
	{"ptyxis", func(d string) []string { return []string{"--working-directory=" + d} }},
	{"x-terminal-emulator", func(d string) []string {
		// debian-alternative — only some honor --working-directory, so chdir via shell.
		return []string{"-e", "sh", "-c", "cd " + shQuote(d) + " && exec ${SHELL:-bash}"}
	}},
	{"xterm", func(d string) []string {
		return []string{"-e", "sh", "-c", "cd " + shQuote(d) + " && exec ${SHELL:-bash}"}
	}},
}

func openTerminal(dir string) {
	// Warp wins if installed — its CLI has no --working-directory flag,
	// but its URI scheme `warp://action/new_tab?path=...` does the job.
	if _, err := exec.LookPath("warp-terminal"); err == nil {
		uri := "warp://action/new_tab?path=" + url.PathEscape(dir)
		cmd := exec.Command("xdg-open", uri)
		if err := cmd.Start(); err == nil {
			go cmd.Wait()
			return
		}
		// fall through to other terminals if xdg-open isn't around
	}
	if env := os.Getenv("TERMINAL"); env != "" {
		if bin, err := exec.LookPath(env); err == nil {
			runTerm(bin, []string{"--working-directory=" + dir}, dir)
			return
		}
	}
	for _, c := range terminalCandidates {
		bin, err := exec.LookPath(c.bin)
		if err != nil {
			continue
		}
		runTerm(bin, c.args(dir), dir)
		return
	}
	fmt.Fprintln(stderr(), "openTerminal: no known terminal found in PATH")
}

func runTerm(bin string, args []string, dir string) {
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir // fallback for terminals whose flags didn't take
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(stderr(), "terminal:", err)
		return
	}
	go cmd.Wait()
}

// shQuote escapes a path for single-quoted shell inclusion.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func addClass(w *gtk.Widget, name string) {
	if ctx, err := w.GetStyleContext(); err == nil {
		ctx.AddClass(name)
	}
}

func loadCSS() error {
	prov, err := gtk.CssProviderNew()
	if err != nil {
		return err
	}
	if err := prov.LoadFromData(styleCSS); err != nil {
		return err
	}
	screen, err := gdk.ScreenGetDefault()
	if err != nil {
		return err
	}
	gtk.AddProviderForScreen(screen, prov, uint(gtk.STYLE_PROVIDER_PRIORITY_APPLICATION))
	return nil
}
