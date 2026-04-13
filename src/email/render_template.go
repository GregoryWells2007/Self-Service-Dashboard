package email

import (
	"bytes"
	"text/template"
)

func RenderTemplate(baseURL string, path string, data any) (string, error) {
	// funcMap := template.FuncMap{
	// 	"asset": func(p string) string {
	// 		return baseURL + "/" + strings.TrimPrefix(p, "/")
	// 	},
	// }

	tmpl := template.Must(template.ParseFiles(path))
	var buf bytes.Buffer
	err := tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
