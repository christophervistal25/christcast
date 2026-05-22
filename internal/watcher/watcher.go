package watcher

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chriscast/chriscast/internal/config"
	"github.com/chriscast/chriscast/internal/index"
	"github.com/chriscast/chriscast/internal/store"
	"github.com/fsnotify/fsnotify"
)

// Watcher recursively watches scope directories and updates the index on
// create/write/remove/rename events. Uses fsnotify (inotify on Linux).
type Watcher struct {
	cfg      *config.Config
	ix       *index.Index
	excludes map[string]struct{}
	w        *fsnotify.Watcher

	mu       sync.Mutex
	watching map[string]struct{}
}

func New(cfg *config.Config, ix *index.Index) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	ex := map[string]struct{}{}
	for _, e := range cfg.Excludes {
		ex[e] = struct{}{}
	}
	return &Watcher{
		cfg:      cfg,
		ix:       ix,
		excludes: ex,
		w:        fw,
		watching: map[string]struct{}{},
	}, nil
}

func (w *Watcher) Close() error { return w.w.Close() }

func (w *Watcher) Run() error {
	for _, s := range w.cfg.Scopes {
		if err := w.addRecursive(s.Path); err != nil {
			log.Printf("watcher: add scope %s: %v", s.Path, err)
		}
	}
	for {
		select {
		case ev, ok := <-w.w.Events:
			if !ok {
				return nil
			}
			w.handle(ev)
		case err, ok := <-w.w.Errors:
			if !ok {
				return nil
			}
			log.Printf("watcher error: %v", err)
		}
	}
}

func (w *Watcher) addRecursive(root string) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if p != root {
			if !w.cfg.IncludeHidden && strings.HasPrefix(name, ".") {
				return fs.SkipDir
			}
			if _, ex := w.excludes[name]; ex {
				return fs.SkipDir
			}
		}
		w.add(p)
		return nil
	})
}

func (w *Watcher) add(dir string) {
	w.mu.Lock()
	if _, ok := w.watching[dir]; ok {
		w.mu.Unlock()
		return
	}
	w.watching[dir] = struct{}{}
	w.mu.Unlock()
	if err := w.w.Add(dir); err != nil {
		log.Printf("watch %s: %v", dir, err)
	}
}

func (w *Watcher) remove(dir string) {
	w.mu.Lock()
	delete(w.watching, dir)
	w.mu.Unlock()
	_ = w.w.Remove(dir)
}

func (w *Watcher) handle(ev fsnotify.Event) {
	name := filepath.Base(ev.Name)
	if !w.cfg.IncludeHidden && strings.HasPrefix(name, ".") {
		return
	}
	if _, ex := w.excludes[name]; ex {
		return
	}
	switch {
	case ev.Op&fsnotify.Create != 0:
		w.onCreate(ev.Name)
	case ev.Op&fsnotify.Write != 0:
		w.onWrite(ev.Name)
	case ev.Op&fsnotify.Remove != 0, ev.Op&fsnotify.Rename != 0:
		w.onRemove(ev.Name)
	}
}

func (w *Watcher) onCreate(p string) {
	info, err := os.Lstat(p)
	if err != nil {
		return
	}
	if info.IsDir() {
		w.add(p)
		w.ix.Upsert(store.FileInfo{
			Path: p, Base: filepath.Base(p),
			ModTime: info.ModTime().Unix(), Size: 0, IsDir: true,
		})
		// scan new subtree
		_ = filepath.WalkDir(p, func(sub string, d fs.DirEntry, err error) error {
			if err != nil || d == nil || sub == p {
				return nil
			}
			fi, ferr := d.Info()
			if ferr != nil {
				return nil
			}
			if d.IsDir() {
				w.add(sub)
				w.ix.Upsert(store.FileInfo{
					Path: sub, Base: d.Name(),
					ModTime: fi.ModTime().Unix(), Size: 0, IsDir: true,
				})
				return nil
			}
			w.ix.Upsert(store.FileInfo{
				Path: sub, Base: d.Name(),
				ModTime: fi.ModTime().Unix(), Size: fi.Size(), IsDir: false,
			})
			return nil
		})
		return
	}
	w.ix.Upsert(store.FileInfo{
		Path: p, Base: filepath.Base(p),
		ModTime: info.ModTime().Unix(), Size: info.Size(), IsDir: false,
	})
}

func (w *Watcher) onWrite(p string) {
	info, err := os.Lstat(p)
	if err != nil {
		return
	}
	if info.IsDir() {
		return
	}
	w.ix.Upsert(store.FileInfo{
		Path: p, Base: filepath.Base(p),
		ModTime: info.ModTime().Unix(), Size: info.Size(), IsDir: false,
	})
}

func (w *Watcher) onRemove(p string) {
	w.ix.Remove(p)
	w.remove(p)
}
