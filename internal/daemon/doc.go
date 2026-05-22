// Package daemon wires the long-running chriscast process together. It
// starts the UI, filesystem watcher, and global hotkey listener, runs a
// periodic index save, and coordinates graceful shutdown on signal. It
// is only compiled when the gtk build tag is set.
package daemon
