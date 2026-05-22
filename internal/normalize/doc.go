// Package normalize applies Unicode NFC normalization and smart-case
// folding to strings used as query inputs and index keys. Smart-case
// preserves case sensitivity when the query contains an uppercase rune
// and folds to lowercase otherwise, keeping query and index keys
// consistent across the pipeline.
package normalize
