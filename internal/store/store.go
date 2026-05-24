package store

type FileID uint32

type FileInfo struct {
	ID      FileID `msgpack:"i"`
	Path    string `msgpack:"p"`
	Base    string `msgpack:"b"`
	ModTime int64  `msgpack:"m"`
	Size    int64  `msgpack:"s"`
	IsDir   bool   `msgpack:"d"`
	IsApp   bool   `msgpack:"a,omitempty"` // .desktop application entry
	Icon    string `msgpack:"icon,omitempty"` // raw Icon= value from .desktop (name or absolute path)
}

type Store struct {
	Files  []FileInfo         `msgpack:"f"`
	byPath map[string]FileID  `msgpack:"-"`
}

func New() *Store { return &Store{byPath: map[string]FileID{}} }

func (s *Store) ensureMap() {
	if s.byPath != nil {
		return
	}
	s.byPath = make(map[string]FileID, len(s.Files))
	for i := range s.Files {
		if s.Files[i].Path != "" {
			s.byPath[s.Files[i].Path] = s.Files[i].ID
		}
	}
}

func (s *Store) Add(fi FileInfo) FileID {
	s.ensureMap()
	fi.ID = FileID(len(s.Files) + 1)
	s.Files = append(s.Files, fi)
	s.byPath[fi.Path] = fi.ID
	return fi.ID
}

func (s *Store) Get(id FileID) *FileInfo {
	if id == 0 || int(id) > len(s.Files) {
		return nil
	}
	fi := &s.Files[id-1]
	if fi.Path == "" {
		return nil
	}
	return fi
}

func (s *Store) IDByPath(p string) (FileID, bool) {
	s.ensureMap()
	id, ok := s.byPath[p]
	return id, ok
}

// Tombstone marks a file as deleted; keeps slot to preserve FileIDs.
func (s *Store) Tombstone(id FileID) *FileInfo {
	if id == 0 || int(id) > len(s.Files) {
		return nil
	}
	s.ensureMap()
	fi := &s.Files[id-1]
	old := *fi
	delete(s.byPath, fi.Path)
	fi.Path = ""
	fi.Base = ""
	return &old
}

func (s *Store) Len() int { return len(s.Files) }
