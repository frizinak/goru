package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/frizinak/goru/data"
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
	r := bytes.NewReader(data.Words)
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
{{ if .Info }}  {{ clrRed }} {{- .Info -}} {{ clrPop }}
{{ end -}}
{{ if .Example }}  {{ .Example }}
{{ if .ExampleTranslation }}  {{ .ExampleTranslation}}
{{ end -}}
{{ end }}
{{- end -}}

{{- define "gender" -}}{{ genderSymbol . }}{{- end -}}

{{- define "word" -}}
{{ clrGreen }} {{- word . -}} {{ clrPop }}
{{- if .NounInfo }} {{ template "gender" .NounInfo.Gender }}{{ end }} {{ .WordType -}}
{{ if .DerivedFrom }} [{{ derived . }}]{{ end }}
{{- range .Translations }}
{{ template "trans" . }}{{ end }}
{{- end -}}

{{- range . }}{{ template "word" . }}
{{ end }}`

	custom := `{{- define "gender" -}}{{ . }}{{- end -}}`

	_clrs := make(clrs, 0)
	clrs := &_clrs
	clrRed := func() clr { return clrs.Get(31) }
	clrGreen := func() clr { return clrs.Get(32) }
	clrYellow := func() clr { return clrs.Get(33) }
	clrBlue := func() clr { return clrs.Get(34) }
	clrMagenta := func() clr { return clrs.Get(35) }
	clrCyan := func() clr { return clrs.Get(36) }
	clrGray := func() clr { return clrs.Get(37) }
	clrPop := func() clr { return clrs.Pop() }

	word := func(w *openrussian.Word) stringer {
		if noStress {
			return strStringer(w.Word)
		}
		p := w.Stressed.Parse()
		list := make(stringList, 0, 5)
		space := strStringer(" ")
		for _, w := range p {
			if w.Stress == "" {
				list = append(list, strStringer(w.Prefix), space)
				continue
			}
			list = append(
				list,
				strStringer(w.Prefix),
				clrYellow(),
				strStringer(w.Stress),
				clrPop(),
				strStringer(w.Suffix),
				space,
			)
		}
		if len(list) != 0 && list[len(list)-1] == space {
			list = list[:len(list)-1]
		}
		return list
	}

	masterTpl, err := template.New("tpls").Funcs(template.FuncMap{
		"derived": func(w *openrussian.Word) stringer {
			l := dict.DerivedList(w)
			s := make(stringList, len(l)*2)
			for i := range l {
				s[i*2] = word(l[i])
				s[i*2+1] = strStringer(" > ")
			}
			if len(s) != 0 {
				return s[:len(s)-1]
			}
			return s
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
		"clrRed":     clrRed,
		"clrGreen":   clrGreen,
		"clrYellow":  clrYellow,
		"clrBlue":    clrBlue,
		"clrMagenta": clrMagenta,
		"clrCyan":    clrCyan,
		"clrGray":    clrGray,
		"clrPop":     clrPop,
		"word":       word,
	}).Parse(master)
	exit(err)

	tpl, err := masterTpl.Parse(custom)
	exit(err)

	results, cyrillic := d.Search(query, all, int(maxResults))
	if len(results) == 0 && cyrillic {
		results = d.SearchRussianFuzzy(query, all, int(maxResults))
	}
	if len(results) == 0 {
		exit(errors.New("no results"))
	}
	exit(tpl.Execute(os.Stdout, results))
}
