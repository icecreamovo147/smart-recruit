//go:build prod

package devlogviewer

import (
	"embed"
	"io/fs"
)

//go:embed web/dist
var embeddedStatic embed.FS

func StaticFS() (fs.FS, error) {
	return fs.Sub(embeddedStatic, "web/dist")
}
