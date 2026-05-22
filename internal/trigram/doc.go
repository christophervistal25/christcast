// Package trigram maintains a padded n=3 inverted index over normalized
// basenames, mapping each trigram to the set of FileIDs that contain it.
// Lookups intersect the per-trigram postings as a multiset to produce
// candidate FileIDs for fuzzy scoring downstream.
package trigram
