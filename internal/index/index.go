package index

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/chriscast/chriscast/internal/apps"
	"github.com/chriscast/chriscast/internal/config"
	"github.com/chriscast/chriscast/internal/normalize"
	"github.com/chriscast/chriscast/internal/scanner"
	"github.com/chriscast/chriscast/internal/store"
	"github.com/chriscast/chriscast/internal/trie"
	"github.com/chriscast/chriscast/internal/trigram"
	"github.com/vmihailenco/msgpack/v5"
)

// Index is the in-memory search index. Only Store is serialized; trie +
// trigram are rebuilt on Load from Store entries — this keeps the on-disk
// format simple and small.
type Index struct {
	Mu      sync.RWMutex
	Store   *store.Store
	Trie    *trie.Trie
	Trigram *trigram.Index
}

// Upsert adds or replaces a file by path. Caller holds no lock; method takes write lock.
func (ix *Index) Upsert(fi store.FileInfo) {
	ix.Mu.Lock()
	defer ix.Mu.Unlock()
	if id, ok := ix.Store.IDByPath(fi.Path); ok {
		old := ix.Store.Get(id)
		if old != nil {
			key := normalize.Fold(old.Base)
			ix.Trie.Delete(key, id)
			ix.Trigram.Delete(key, id)
		}
		ix.Store.Tombstone(id)
	}
	id := ix.Store.Add(fi)
	key := normalize.Fold(fi.Base)
	ix.Trie.Insert(key, id)
	ix.Trigram.Insert(key, id)
}

// Remove removes a file by path.
func (ix *Index) Remove(path string) {
	ix.Mu.Lock()
	defer ix.Mu.Unlock()
	id, ok := ix.Store.IDByPath(path)
	if !ok {
		return
	}
	fi := ix.Store.Get(id)
	if fi == nil {
		return
	}
	key := normalize.Fold(fi.Base)
	ix.Trie.Delete(key, id)
	ix.Trigram.Delete(key, id)
	ix.Store.Tombstone(id)
}

func New() *Index {
	return &Index{
		Store:   store.New(),
		Trie:    trie.New(),
		Trigram: trigram.New(),
	}
}

type Progress func(n int)

func (ix *Index) Build(c *config.Config, p Progress) (scanner.Stats, error) {
	start := time.Now()
	st, err := scanner.Scan(c, func(fi store.FileInfo) {
		ix.Mu.Lock()
		id := ix.Store.Add(fi)
		key := normalize.Fold(fi.Base)
		ix.Trie.Insert(key, id)
		ix.Trigram.Insert(key, id)
		n := ix.Store.Len()
		ix.Mu.Unlock()
		if p != nil && n%5000 == 0 {
			p(n)
		}
	})
	if err != nil {
		return st, err
	}
	// Index installed XDG applications alongside files.
	for _, fi := range apps.Scan() {
		ix.Upsert(fi)
	}
	_ = start
	return st, nil
}

func IndexPath() string { return filepath.Join(config.DataDir(), "index.msgpack") }

// blob is the on-disk shape. Only Store is persisted; secondary structures
// are rebuilt at Load time.
type blob struct {
	Version int          `msgpack:"v"`
	Store   *store.Store `msgpack:"s"`
}

func (ix *Index) Save() error {
	if err := os.MkdirAll(config.DataDir(), 0o755); err != nil {
		return err
	}
	tmp := IndexPath() + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	ix.Mu.RLock()
	enc := msgpack.NewEncoder(f)
	encErr := enc.Encode(blob{Version: 1, Store: ix.Store})
	ix.Mu.RUnlock()
	if encErr != nil {
		f.Close()
		return encErr
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, IndexPath())
}

func Load() (*Index, error) {
	f, err := os.Open(IndexPath())
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var b blob
	if err := msgpack.NewDecoder(f).Decode(&b); err != nil {
		return nil, fmt.Errorf("decode index: %w", err)
	}
	if b.Version != 1 {
		return nil, fmt.Errorf("unsupported index version: %d", b.Version)
	}
	if b.Store == nil {
		return nil, fmt.Errorf("corrupted index: missing store")
	}
	ix := &Index{Store: b.Store, Trie: trie.New(), Trigram: trigram.New()}
	for _, fi := range b.Store.Files {
		key := normalize.Fold(fi.Base)
		ix.Trie.Insert(key, fi.ID)
		ix.Trigram.Insert(key, fi.ID)
	}
	return ix, nil
}
