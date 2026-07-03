package master

import "github.com/trusted-technologies/cuttlefish/internal/web"

func init() {
	staticFS = web.StaticFS
}
