package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/frizinak/goru/bound"
	"github.com/frizinak/goru/dict"
	"github.com/frizinak/goru/openrussian"
)

func exit(err error) {
	if err == nil {
		return
	}

	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func getDict() (*dict.Dict, error) {
	a, err := bound.Asset("db.gob")
	if err != nil {
		return nil, err
	}
	r := bytes.NewReader(a)
	words, err := openrussian.DecodeGOB(r)
	if err != nil {
		return nil, err
	}
	return dict.New(words), nil
}

func main() {
	var maxResults uint
	var all bool
	var noStress bool
	flag.UintVar(&maxResults, "n", 3, "max amount of results")
	flag.BoolVar(&all, "a", false, "include words without translation")
	flag.BoolVar(&noStress, "ns", false, "don't print stress mark")
	flag.Parse()

	query := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if query == "" {
		exit(errors.New("please provide a query"))
	}

	d, err := getDict()
	exit(err)

	master := `{{- define "trans" }}  {{ .Translation }}
{{ if .Info }}  {{ clrRed }} {{- .Info -}} {{ clrReset }}
{{ end -}}
{{ if .Example }}  {{ .Example }}
{{ if .ExampleTranslation }}  {{ .ExampleTranslation}}
{{ end -}}
{{ end }}
{{- end -}}

{{- define "gender" -}}{{ genderSymbol . }}{{- end -}}

{{- define "word" -}}
{{ clrGreen }} {{- word . -}} {{ clrReset }}
{{- if .NounInfo }} {{ template "gender" .NounInfo.Gender }}{{ end }} {{ .WordType -}}
{{ if .DerivedFrom }} [{{ word .DerivedFrom }}]{{ end }}
{{- range .Translations }}
{{ template "trans" . }}{{ end }}
{{- end -}}

{{- range . }}{{ template "word" . }}
{{ end }}`

	custom := `{{- define "gender" -}}{{ . }}{{- end -}}`

	word := func(w *openrussian.Word) string {
		if noStress {
			return w.Word
		}
		return string(w.Stressed)
	}

	masterTpl, err := template.New("tpls").Funcs(template.FuncMap{
		"derived": func(w *openrussian.Word) string {
			l := dict.DerivedList(w)
			s := make([]string, len(l))
			for i := range l {
				s[i] = word(l[i])
			}
			return strings.Join(s, " > ")
		},
		"genderSymbol": func(g openrussian.Gender) string {
			switch g {
			case openrussian.N:
				return "⚲"
			case openrussian.F:
				return "♀"
			case openrussian.M:
				return "♂"
			}

			return "?"
		},
		"clrRed":   func() string { return "\033[31m" },
		"clrGreen": func() string { return "\033[32m" },
		"clrReset": func() string { return "\033[0m" },
		"word":     word,
	}).Parse(master)
	exit(err)
	tpl, err := masterTpl.Parse(custom)
	exit(err)
	results := d.Search(query, all, int(maxResults))
	exit(tpl.Execute(os.Stdout, results))
	// for _, r := range results {
	// 	//exit(tpls.ExecuteTemplate(os.Stdout, "word", r))
	// 	exit(tpls.Execute(os.Stdout, r))
	// 	//fmt.Println(r.StringWithTranslations(true, true))
	// 	fmt.Println()
	// }

}
