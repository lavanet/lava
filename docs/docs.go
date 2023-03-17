package docs

import (
	"embed"
	httptemplate "html/template"
	"net/http"

	"github.com/gorilla/mux"
)

const (
	apiFilePath       = "/static/openapi.yml"
	indexTemplatePath = "template/index.tpl"
)

//go:embed static
var staticFiles embed.FS

//go:embed template
var templateFiles embed.FS

func RegisterOpenAPIService(appName string, rtr *mux.Router) {
	rtr.Handle(apiFilePath, http.FileServer(http.FS(staticFiles)))
	rtr.HandleFunc("/", handler(appName))
}

// openapi handler returns a handler that serves the openapi documentation
func handler(title string) http.HandlerFunc {
	t, err := httptemplate.ParseFS(templateFiles, indexTemplatePath)
	if err != nil {
		panic(err)
	}

	return func(w http.ResponseWriter, req *http.Request) {
		t.Execute(w, struct {
			Title string
			URL   string
		}{
			title,
			apiFilePath,
		})
	}
}
