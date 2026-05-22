//go:build gtk

package main

import (
	"os"

	"github.com/chriscast/chriscast/internal/config"
	"github.com/chriscast/chriscast/internal/daemon"
	"github.com/chriscast/chriscast/internal/index"
)

func runDaemon() {
	c, err := config.Load()
	if err != nil {
		die("config: %v", err)
	}
	ix, err := index.Load()
	if err != nil {
		// no existing index — build now
		ix = index.New()
		st, berr := ix.Build(c, nil)
		if berr != nil {
			die("build index: %v", berr)
		}
		_ = st
		if serr := ix.Save(); serr != nil {
			die("save index: %v", serr)
		}
	}
	d, err := daemon.New(c, ix)
	if err != nil {
		die("daemon: %v", err)
	}
	if err := d.Run(); err != nil {
		die("daemon: %v", err)
	}
	os.Exit(0)
}
