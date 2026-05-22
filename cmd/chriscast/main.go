package main

import (
	"fmt"
	"os"
	"time"

	"github.com/chriscast/chriscast/internal/config"
	"github.com/chriscast/chriscast/internal/index"
	"github.com/chriscast/chriscast/internal/search"
	"github.com/chriscast/chriscast/internal/ui"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "index", "reindex":
		runIndex()
	case "search":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "search: missing query")
			os.Exit(2)
		}
		runSearch(os.Args[2])
	case "info":
		runInfo()
	case "ui":
		runUI()
	case "daemon":
		runDaemon()
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `chriscast — file launcher (Phase 1)

usage:
  chriscast index            build/rebuild the file index
  chriscast reindex          alias for index
  chriscast search "query"   print top results
  chriscast ui               launch desktop overlay (one-shot)
  chriscast daemon           background: watch files + global hotkey
  chriscast info             show index stats
  chriscast help             this message`)
}

func runIndex() {
	c, err := config.Load()
	if err != nil {
		die("config: %v", err)
	}
	ix := index.New()
	start := time.Now()
	st, err := ix.Build(c, func(n int) {
		fmt.Fprintf(os.Stderr, "\rindexed %d files...", n)
	})
	if err != nil {
		die("\nbuild: %v", err)
	}
	if err := ix.Save(); err != nil {
		die("\nsave: %v", err)
	}
	fmt.Fprintf(os.Stderr, "\rindexed %d files, %d dirs, %d skipped in %s\n",
		st.Files, st.Dirs, st.Skipped, time.Since(start).Round(time.Millisecond))
	fmt.Fprintf(os.Stderr, "index at %s\n", index.IndexPath())
}

func runSearch(q string) {
	c, err := config.Load()
	if err != nil {
		die("config: %v", err)
	}
	ix, err := index.Load()
	if err != nil {
		die("load index: %v (run `chriscast index` first)", err)
	}
	results := search.Search(ix, q, c.MaxResults)
	for _, r := range results {
		suffix := ""
		if r.File.IsDir {
			suffix = "/"
		}
		fmt.Printf("%5d  %s%s\n", r.Score, r.File.Path, suffix)
	}
}

func runUI() {
	c, err := config.Load()
	if err != nil {
		die("config: %v", err)
	}
	ix, err := index.Load()
	if err != nil {
		die("load index: %v (run `chriscast index` first)", err)
	}
	if err := ui.Run(c, ix); err != nil {
		die("ui: %v", err)
	}
}

func runInfo() {
	ix, err := index.Load()
	if err != nil {
		die("load index: %v", err)
	}
	fmt.Printf("index path:  %s\n", index.IndexPath())
	fmt.Printf("entries:     %d\n", ix.Store.Len())
	fmt.Printf("config path: %s\n", config.Path())
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
