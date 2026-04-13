package email

import (
	"bytes"
	"path/filepath"
	"text/template"
)

func RenderTemplate(path string, data any, funcMap template.FuncMap) (string, error) {
	tmpl := template.New("")

	if funcMap != nil {
		tmpl = tmpl.Funcs(funcMap)
	}

	tmpl, err := tmpl.ParseFiles(path)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, filepath.Base(path), data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
