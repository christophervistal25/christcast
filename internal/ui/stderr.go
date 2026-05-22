//go:build gtk

package ui

import (
	"io"
	"os"
)

func stderr() io.Writer { return os.Stderr }
