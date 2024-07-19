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

func RegisterOpenAPIService(appName string, rtr *mux.Router) error {
	if err := registerStaticFiles(rtr); err != nil {
		return err
	}
	if err := registerTemplateHandler(appName, rtr); err != nil {
		return err
	}
	return nil
}

func registerStaticFiles(rtr *mux.Router) error {
	rtr.Handle(apiFilePath, http.FileServer(http.FS(staticFiles)))
	return nil
}

func registerTemplateHandler(appName string, rtr *mux.Router) error {
	t, err := parseTemplates()
	if err != nil {
		return err
	}
	rtr.HandleFunc("/", handler(t, appName))
	return nil
}

func parseTemplates() (*httptemplate.Template, error) {
	t, err := httptemplate.ParseFS(templateFiles, indexTemplatePath)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func handler(t *httptemplate.Template, title string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		data := struct {
			Title string
			URL   string
		}{
			Title: title,
			URL:   apiFilePath,
		}
		if err := t.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

