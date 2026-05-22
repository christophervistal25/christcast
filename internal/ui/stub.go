//go:build !gtk

package ui

import (
	"errors"

	"github.com/chriscast/chriscast/internal/config"
	"github.com/chriscast/chriscast/internal/index"
)

func Run(_ *config.Config, _ *index.Index) error {
	return errors.New("UI not compiled — rebuild with `-tags gtk` (requires libgtk-3-dev)")
}
