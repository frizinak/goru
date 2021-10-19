package common

import (
	"bytes"
	htmltpl "html/template"
	"text/template"

	"github.com/frizinak/goru/data"
	"github.com/frizinak/goru/dict"
	"github.com/frizinak/goru/openrussian"
)

const tplStr = `{{- define "trans" }}  {{ .Translation }}
{{ if .Info }}  {{ clrRed }} {{- .Info -}} {{ clrPop }}
{{ end -}}
{{ if .Example }}  {{ .Example }}
{{ if .ExampleTranslation }}  {{ .ExampleTranslation}}
{{ end -}}
{{ end }}
{{- end -}}

{{- define "gender" -}}{{ genderSymbol . }}{{- end -}}

{{- define "word" -}}
{{ template "wordStr" . }}
{{- if .NounInfo }} {{ template "gender" .NounInfo.Gender }}{{ end }} {{ .WordType -}}
{{ if .DerivedFrom }} [{{ derived . }}]{{ end }}
{{- range .Translations }}
{{ template "trans" . }}{{ end }}
{{- end -}}

{{- define "wordStr" -}}
{{ clrGreen }} {{- stressed . -}} {{ clrPop }}
{{- end -}}

{{- range . }}{{ template "word" . }}
{{ end }}`

var dct *dict.Dict
var tpl *template.Template
var httpl *htmltpl.Template

func GetDict() (*dict.Dict, error) {
	if dct != nil {
		return dct, nil
	}
	r := bytes.NewReader(data.Words)
	words, err := openrussian.DecodeGOB(r)
	if err != nil {
		return nil, err
	}

	dct = dict.New(words)
	return dct, nil
}

func getTplFuncs() template.FuncMap {
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

	unstressed := func(w *openrussian.Word) stringer {
		return strStringer(w.Word)
	}

	stressednc := func(w *openrussian.Word) stringer {
		return strStringer(w.Stressed.Parse().String())
	}

	stressed := func(w *openrussian.Word) stringer {
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

	return template.FuncMap{
		"derived": func(w *openrussian.Word) stringer {
			l := dict.DerivedList(w)
			s := make(stringList, len(l)*2)
			for i := range l {
				s[i*2] = unstressed(l[i])
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
		"stressed":   stressed,
		"unstressed": unstressed,
		"stressednc": stressednc,
	}
}

func GetHTMLTpl() (*htmltpl.Template, error) {
	if httpl != nil {
		return httpl, nil
	}

	var err error
	httpl, err = htmltpl.New("tpls").Funcs(htmltpl.FuncMap(getTplFuncs())).Parse(tplStr)

	return httpl, err
}

func GetTpl() (*template.Template, error) {
	if tpl != nil {
		return tpl, nil
	}

	var err error
	tpl, err = template.New("tpls").Funcs(getTplFuncs()).Parse(tplStr)

	return tpl, err
}
