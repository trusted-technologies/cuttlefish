package master

import (
	"io/fs"

	"github.com/trusted-technologies/cuttlefish/internal/web"
)

func init() {
	var err error
	staticFS, err = fs.Sub(web.StaticFS, "static")
	if err != nil {
		panic(err)
	}
}
