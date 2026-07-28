package server

import (
	"errors"
	"io/fs"
)

// isNotExist reports whether err is a not-exist-style error.
func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}