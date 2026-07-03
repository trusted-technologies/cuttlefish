package master

import (
	"html/template"

	"github.com/trusted-technologies/cuttlefish/internal/web"
)

// LoadTemplates parses the embedded HTML templates.
func LoadTemplates() (*template.Template, error) {
	return template.ParseFS(web.TemplatesFS, "templates/*.html")
}

func init() {
	staticFS = web.StaticFS
}
