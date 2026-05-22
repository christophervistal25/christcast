// Package index combines the Store, Trie, and Trigram structures behind
// a single RWMutex-guarded type. It exposes Build, Save, and Load using
// msgpack for persistence, along with Upsert and Remove operations that
// keep all three substructures consistent under concurrent access.
package index
