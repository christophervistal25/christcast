// Package scanner walks configured root directories in parallel under a
// bounded semaphore and emits FileInfo events for each discovered entry.
// It feeds the initial index build and any later full rescans.
package scanner
