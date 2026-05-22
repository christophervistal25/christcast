package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/chriscast/chriscast/internal/config"
	"github.com/chriscast/chriscast/internal/store"
)

type Stats struct {
	Files   int64
	Dirs    int64
	Skipped int64
}

type Visitor func(store.FileInfo)

// Scan walks all scopes in parallel using a semaphore-bounded
// goroutine-per-directory pattern. visit is serialized via mutex so
// callers don't need to be thread-safe.
func Scan(c *config.Config, visit Visitor) (Stats, error) {
	var files, dirs, skipped int64
	excludes := map[string]struct{}{}
	for _, e := range c.Excludes {
		excludes[e] = struct{}{}
	}

	parallelism := runtime.NumCPU() * 2
	if parallelism < 4 {
		parallelism = 4
	}
	sem := make(chan struct{}, parallelism)
	var wg sync.WaitGroup
	var visitMu sync.Mutex

	var walk func(dir string, rootDev uint64)
	walk = func(dir string, rootDev uint64) {
		sem <- struct{}{}
		defer func() {
			<-sem
			wg.Done()
		}()

		atomic.AddInt64(&dirs, 1)
		entries, err := os.ReadDir(dir)
		if err != nil {
			atomic.AddInt64(&skipped, 1)
			return
		}
		for _, e := range entries {
			name := e.Name()
			if !c.IncludeHidden && strings.HasPrefix(name, ".") {
				continue
			}
			if _, ex := excludes[name]; ex {
				continue
			}
			p := filepath.Join(dir, name)
			if e.IsDir() {
				if !c.FollowSymlinks && e.Type()&fs.ModeSymlink != 0 {
					continue
				}
				if !c.CrossDevice {
					if dev, ok := deviceOf(p); ok && dev != rootDev {
						continue
					}
				}
				// emit the directory itself as a searchable entry
				if info, ierr := e.Info(); ierr == nil {
					visitMu.Lock()
					visit(store.FileInfo{
						Path:    p,
						Base:    name,
						ModTime: info.ModTime().Unix(),
						Size:    0,
						IsDir:   true,
					})
					visitMu.Unlock()
				}
				wg.Add(1)
				go walk(p, rootDev)
				continue
			}
			// skip symlinked files too (we already skip symlink dirs above)
			if !c.FollowSymlinks && e.Type()&fs.ModeSymlink != 0 {
				continue
			}
			info, ierr := e.Info()
			if ierr != nil {
				atomic.AddInt64(&skipped, 1)
				continue
			}
			atomic.AddInt64(&files, 1)
			fi := store.FileInfo{
				Path:    p,
				Base:    name,
				ModTime: info.ModTime().Unix(),
				Size:    info.Size(),
				IsDir:   false,
			}
			visitMu.Lock()
			visit(fi)
			visitMu.Unlock()
		}
	}

	for _, s := range c.Scopes {
		dev, _ := deviceOf(s.Path)
		wg.Add(1)
		go walk(s.Path, dev)
	}
	wg.Wait()

	return Stats{Files: files, Dirs: dirs, Skipped: skipped}, nil
}

func deviceOf(p string) (uint64, bool) {
	var st syscall.Stat_t
	if err := syscall.Stat(p, &st); err != nil {
		return 0, false
	}
	return uint64(st.Dev), true
}
