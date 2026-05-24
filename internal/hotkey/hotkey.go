//go:build gtk

package hotkey

import (
	"fmt"
	"strings"

	"github.com/jezek/xgb/xproto"
	"github.com/jezek/xgbutil"
	"github.com/jezek/xgbutil/keybind"
	"github.com/jezek/xgbutil/xevent"
)

// Listener grabs a global hotkey via X11 XGrabKey and fires cb on press.
type Listener struct {
	x   *xgbutil.XUtil
	cb  func(timestamp uint32)
	mod uint16
	kc  xproto.Keycode
}

func New(combo string, cb func(timestamp uint32)) (*Listener, error) {
	x, err := xgbutil.NewConn()
	if err != nil {
		return nil, fmt.Errorf("X conn: %w", err)
	}
	keybind.Initialize(x)
	mods, kcs, err := keybind.ParseString(x, normalize(combo))
	if err != nil {
		return nil, fmt.Errorf("parse hotkey %q: %w", combo, err)
	}
	if len(kcs) == 0 {
		return nil, fmt.Errorf("no keycodes for %q", combo)
	}
	l := &Listener{x: x, cb: cb, mod: mods, kc: kcs[0]}
	root := x.RootWin()
	if err := keybind.GrabChecked(x, root, mods, kcs[0]); err != nil {
		return nil, fmt.Errorf("grab: %w", err)
	}
	xevent.KeyPressFun(func(_ *xgbutil.XUtil, e xevent.KeyPressEvent) {
		cb(uint32(e.Time))
	}).Connect(x, root)
	return l, nil
}

func (l *Listener) Run() { xevent.Main(l.x) }

func (l *Listener) Close() {
	keybind.Ungrab(l.x, l.x.RootWin(), l.mod, l.kc)
	l.x.Conn().Close()
}

// normalize converts our "ctrl+space" → keybind's "Control-space" form.
func normalize(s string) string {
	s = strings.ToLower(s)
	parts := strings.Split(s, "+")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		switch p {
		case "ctrl", "control":
			out = append(out, "Control")
		case "shift":
			out = append(out, "Shift")
		case "alt", "mod1":
			out = append(out, "Mod1")
		case "super", "meta", "mod4":
			out = append(out, "Mod4")
		default:
			out = append(out, p)
		}
	}
	return strings.Join(out, "-")
}
