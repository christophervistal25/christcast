// Package hotkey registers a global X11 hotkey via xgbutil's XGrabKey
// and delivers press events to the daemon so the overlay can be toggled
// from anywhere. It is only compiled when the gtk build tag is set.
package hotkey
