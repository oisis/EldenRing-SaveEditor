package dbviewer

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"
)

//go:embed web/templates/*.html web/assets/*
var webFiles embed.FS

type templateSet struct {
	templates *template.Template
}

func parseTemplates() (templateSet, error) {
	templates, err := template.New("dbviewer").
		Funcs(template.FuncMap{"sourceOrigin": sourceOriginClass}).
		ParseFS(webFiles, "web/templates/*.html")
	if err != nil {
		return templateSet{}, fmt.Errorf("parse viewer templates: %w", err)
	}
	return templateSet{templates: templates}, nil
}

func (server *Server) render(response http.ResponseWriter, name string, data any) {
	var rendered bytes.Buffer
	if err := server.templates.templates.ExecuteTemplate(&rendered, name, data); err != nil {
		http.Error(response, "Unable to render database view.", http.StatusInternalServerError)
		return
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = response.Write(rendered.Bytes())
}
