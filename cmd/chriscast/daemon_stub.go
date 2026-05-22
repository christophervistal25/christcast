//go:build !gtk

package main

func runDaemon() {
	die("daemon requires GTK build — rebuild with `make build-ui`")
}
