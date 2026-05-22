package search

import (
	"container/heap"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chriscast/chriscast/internal/index"
	"github.com/chriscast/chriscast/internal/normalize"
	"github.com/chriscast/chriscast/internal/score"
	"github.com/chriscast/chriscast/internal/store"
)

type Result struct {
	File  *store.FileInfo
	Score int
}

const (
	candidateLimit = 5000
	prefixCap      = 2000
)

// extension biases.
var (
	demoteExt = map[string]int{".log": -20, ".tmp": -20, ".bak": -10, ".swp": -30, ".pyc": -15}
	boostExt  = map[string]int{".md": 4, ".txt": 3, ".go": 4, ".ts": 3, ".tsx": 3, ".js": 2, ".py": 4, ".pdf": 3}
)

// Search returns top results for q against the basename index.
// If q contains '/', secondary path-token scoring is also applied.
func Search(ix *index.Index, q string, limit int) []Result {
	if q == "" || ix == nil {
		return nil
	}
	ix.Mu.RLock()
	defer ix.Mu.RUnlock()

	// Directory-browse mode: query is an absolute path to an existing dir.
	// e.g. typing "/var/www/html" lists its direct children.
	if strings.HasPrefix(q, "/") {
		dir := strings.TrimRight(q, "/")
		if dir == "" {
			dir = "/"
		}
		if id, ok := ix.Store.IDByPath(dir); ok {
			if fi := ix.Store.Get(id); fi != nil && fi.IsDir {
				return listChildren(ix, dir, limit)
			}
		}
		// fallback: not in index, but path exists on disk — readdir live.
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			return liveReaddir(dir, limit)
		}
	}

	lookup, scoreQ, caseSensitive := normalize.SmartCaseQuery(q)
	pathQuery := strings.Contains(lookup, "/")

	seen := map[store.FileID]struct{}{}
	add := func(ids []store.FileID) {
		for _, id := range ids {
			seen[id] = struct{}{}
		}
	}
	// 1. trie prefix walk — basename-key index, lowercase
	add(ix.Trie.PrefixIDs(stripLeadingSlash(lookup), prefixCap))
	// 2. trigram intersection over basename
	last := lookup
	if pathQuery {
		if i := strings.LastIndex(lookup, "/"); i >= 0 {
			last = lookup[i+1:]
		}
	}
	if len(last) >= 1 {
		add(ix.Trigram.Candidates(last, candidateLimit))
	}

	now := time.Now().Unix()
	h := &resultHeap{}
	heap.Init(h)
	for id := range seen {
		fi := ix.Store.Get(id)
		if fi == nil {
			continue
		}
		s := scoreFile(fi, scoreQ, caseSensitive, pathQuery)
		if s <= 0 {
			continue
		}
		s += recencyBoost(fi.ModTime, now)
		s += extBias(fi.Base)
		if fi.IsDir {
			s += 2 // slight nudge so dirs aren't buried beneath same-name files
		}
		heap.Push(h, Result{File: fi, Score: s})
		if h.Len() > limit {
			heap.Pop(h)
		}
	}
	out := make([]Result, h.Len())
	for i := len(out) - 1; i >= 0; i-- {
		out[i] = heap.Pop(h).(Result)
	}
	return out
}

// liveReaddir reads a directory live (no index) — used when user types
// a path outside indexed scope. Returns entries with dirs first.
func liveReaddir(dir string, limit int) []Result {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var dirs, files []Result
	// caller doesn't own these — use a slice we own so &fi[i] is stable.
	stored := make([]store.FileInfo, 0, len(entries))
	for _, e := range entries {
		info, ierr := e.Info()
		if ierr != nil {
			continue
		}
		size := int64(0)
		if !e.IsDir() {
			size = info.Size()
		}
		stored = append(stored, store.FileInfo{
			Path:    filepath.Join(dir, e.Name()),
			Base:    e.Name(),
			ModTime: info.ModTime().Unix(),
			Size:    size,
			IsDir:   e.IsDir(),
		})
	}
	for i := range stored {
		r := Result{File: &stored[i], Score: 1}
		if stored[i].IsDir {
			dirs = append(dirs, r)
		} else {
			files = append(files, r)
		}
	}
	out := append(dirs, files...)
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// listChildren returns direct children of dir from the index, dirs first.
func listChildren(ix *index.Index, dir string, limit int) []Result {
	var dirs, files []Result
	for i := range ix.Store.Files {
		fi := &ix.Store.Files[i]
		if fi.Path == "" {
			continue
		}
		if filepath.Dir(fi.Path) != dir {
			continue
		}
		r := Result{File: fi, Score: 1}
		if fi.IsDir {
			dirs = append(dirs, r)
		} else {
			files = append(files, r)
		}
	}
	out := append(dirs, files...)
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func scoreFile(fi *store.FileInfo, query string, caseSensitive, pathQuery bool) int {
	base := fi.Base
	if !caseSensitive {
		base = normalize.Fold(base)
	}
	q := query
	if pathQuery {
		// score against full path with path-component tokenization preserved.
		path := fi.Path
		if !caseSensitive {
			path = normalize.Fold(path)
		}
		return score.Match(path, q)
	}
	return score.Match(base, q)
}

func recencyBoost(modUnix, nowUnix int64) int {
	delta := nowUnix - modUnix
	if delta < 0 {
		delta = 0
	}
	days := float64(delta) / 86400.0
	// fresher files get larger boost; clamp to [0,10]
	b := 10.0 - math.Log2(1.0+days)
	if b < 0 {
		return 0
	}
	if b > 10 {
		b = 10
	}
	return int(b)
}

func extBias(base string) int {
	i := strings.LastIndex(base, ".")
	if i < 0 {
		return 0
	}
	ext := strings.ToLower(base[i:])
	if v, ok := demoteExt[ext]; ok {
		return v
	}
	if v, ok := boostExt[ext]; ok {
		return v
	}
	return 0
}

func stripLeadingSlash(s string) string {
	for len(s) > 0 && s[0] == '/' {
		s = s[1:]
	}
	return s
}

// heap of results — min-heap so smallest score pops first; we keep top K.
type resultHeap []Result

func (h resultHeap) Len() int           { return len(h) }
func (h resultHeap) Less(i, j int) bool { return h[i].Score < h[j].Score }
func (h resultHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *resultHeap) Push(x any)        { *h = append(*h, x.(Result)) }
func (h *resultHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}
