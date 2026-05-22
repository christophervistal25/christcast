// Package store defines the canonical FileID and FileInfo types and an
// in-memory Store that maps absolute paths to compact integer IDs. The
// Store supports tombstoning so that removed entries can be skipped
// without invalidating IDs held elsewhere in the pipeline.
package store
