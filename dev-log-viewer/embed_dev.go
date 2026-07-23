//go:build !prod

package devlogviewer

import (
	"io/fs"
	"os"
	"path/filepath"
)

func StaticFS() (fs.FS, error) {
	return os.DirFS(filepath.Join("web", "dist")), nil
}
