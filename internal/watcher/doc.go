// Package watcher uses fsnotify to monitor the configured roots
// recursively and forwards create, write, rename, and remove events into
// the Index as live Upsert and Remove calls. It keeps the index in sync
// between scheduled saves without requiring a full rescan.
package watcher
