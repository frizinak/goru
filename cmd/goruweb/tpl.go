package main

import (
	"strings"

	"github.com/frizinak/goru/data"
)

func nonl(i string) string { return strings.ReplaceAll(strings.ReplaceAll(i, "\n", ""), "\t", "") }

var mainTpl = nonl(`{{- define "trans" -}}
<div>{{ .Translation }}</div>
{{- if .Info }}<div>{{ .Info }}</div>{{ end -}}
{{- if .Example -}}
<div>
	<p>{{ .Example -}}</p>
	{{ if .ExampleTranslation }}<p>{{ .ExampleTranslation}}</p>{{ end -}}
</div>
{{- end -}}
{{- end -}}

{{- define "gender" -}}{{ with genderImg . }}<img class="gender" src="{{ . }}"/>{{ end }}{{- end -}}

{{- define "arb-img" -}}
<td class="img-container">
{{- if . -}}
{{- $img := absArbitraryImg .String -}}
{{- $aud := absArbitraryAudio .Unstressed -}}
<div class="scrape">
<a href="{{ $img }}">img</a>
<!--<a href="{{ $aud }}">audio</a>-->
</div>
<a class="img" href="#"><img src="{{ $img }}"/></a>
<audio controls>
<source src="{{ $aud }}" type="audio/mpeg">
</audio>
{{- end -}}
</td>
{{- end -}}

{{- define "arb-img-word" -}}
<td class="img-container">{{ if . }}{{ if .Stressed }}{{- template "arb-img" .Stressed -}}{{ end }}{{ end }}</td>
{{- end -}}

{{- define "word-info" -}}
<div class="meta">
{{- with .AdjInfo -}}
	<table class="adj">
		{{- with .Comparative -}}
		<tr><td>Comparative</td><td>{{ .Unstressed }}</td>{{ template "arb-img" . }}</tr>
		{{- end -}}
		{{- with .Superlative -}}
		<tr><td>Superlative</td><td>{{ .Unstressed }}</td>{{ template "arb-img" . }}</tr>
		{{- end -}}
		<tr><td></td><td></td></tr>
		{{ with .F }}{{ template "adj-gender-info" . }}{{ end -}}
		<tr><td></td><td></td></tr>
		{{ with .M }}{{ template "adj-gender-info" . }}{{ end -}}
		<tr><td></td><td></td></tr>
		{{ with .N }}{{ template "adj-gender-info" . }}{{ end -}}
		<tr><td></td><td></td></tr>
		{{ with .Pl }}{{ template "adj-gender-info" . }}{{ end -}}
	</table>
{{- end -}}
{{- with .VerbInfo -}}
	<table class="verb">
	{{- with .Conjugation -}}
		<tr><td>я</td><td>{{ .Sg1.Unstressed }}</td>{{ template "arb-img" .Sg1 }}</td>
		<tr><td>ты</td><td>{{ .Sg2.Unstressed }}</td>{{ template "arb-img" .Sg2 }}</td>
		<tr><td>он/она/оно</td><td>{{ .Sg3.Unstressed }}</td>{{ template "arb-img" .Sg3 }}</td>
		<tr><td>мы</td><td>{{ .Pl1.Unstressed }}</td>{{ template "arb-img" .Pl1 }}</td>
		<tr><td>вы</td><td>{{ .Pl2.Unstressed }}</td>{{ template "arb-img" .Pl2 }}</td>
		<tr><td>они</td><td>{{ .Pl3.Unstressed }}</td>{{ template "arb-img" .Pl3 }}</td>
		<tr><td></td><td></td></tr>
	{{ end -}}
		<tr><td>Aspect</td><td>{{ .Aspect }}</td></tr>
		{{- with .ImperativeSg }}<tr><td>Imperative (singular)</td><td>{{ .Unstressed }}</td>{{ template "arb-img" . }}</tr>{{ end -}}
		{{- with .ImperativePl }}<tr><td>Imperative (plural)</td><td>{{ .Unstressed }}</td>{{ template "arb-img" . }}</tr>{{ end -}}
		<tr><td></td><td></td></tr>
		{{- with .PastF }}<tr><td>past {{ template "gender" "f" }}</td><td>{{ .Unstressed }}</td>{{ template "arb-img" . }}</tr>{{ end -}}
		{{- with .PastM }}<tr><td>past {{ template "gender" "m" }}</td><td>{{ .Unstressed }}</td>{{ template "arb-img" . }}</tr>{{ end -}}
		{{- with .PastN }}<tr><td>past {{ template "gender" "n" }}</td><td>{{ .Unstressed }}</td>{{ template "arb-img" . }}</tr>{{ end -}}
		{{- with .PastPl }}<tr><td>past {{ template "gender" "pl" }}</td><td>{{ .Unstressed }}</td>{{ template "arb-img" . }}</tr>{{ end -}}
		<tr><td></td><td></td></tr>
		<tr><td>Active present</td><td>{{ with .ActivePresent }}{{ .Word }}{{ else }}/{{ end }}</td>{{ template "arb-img-word" .ActivePresent }}</tr>
		<tr><td>Active past</td><td>{{ with .ActivePast }}{{ .Word }}{{ else }}/{{ end }}</td>{{ template "arb-img-word" .ActivePast }}</tr>
		<tr><td>Passive present</td><td>{{ with .PassivePresent }}{{ .Word }}{{ else }}/{{ end }}</td>{{ template "arb-img-word" .PassivePresent }}</tr>
		<tr><td>Passive past</td><td>{{ with .PassivePast }}{{ .Word }}{{ else }}/{{ end }}</td>{{ template "arb-img-word" .PassivePast }}</tr>
		<tr><td></td><td></td></tr>
	</table>
{{- end -}}
</div>
{{- end -}}

{{- define "adj-gender-info" -}}
<tr><td class="title">{{ template "gender" .Gender }}</td><td></td></tr>
{{- with .Short }}<tr><td>stem</td><td>{{ .Unstressed }}</td></tr>{{ end -}}
{{- with .Decl -}}
	{{- with .Nom }}<tr><td>nom</td><td>{{ .Unstressed }}</td>{{ template "arb-img" . }}</tr>{{ end -}}
	{{- with .Gen }}<tr><td>gen</td><td>{{ .Unstressed }}</td>{{ template "arb-img" . }}</tr>{{ end -}}
	{{- with .Dat }}<tr><td>dat</td><td>{{ .Unstressed }}</td>{{ template "arb-img" . }}</tr>{{ end -}}
	{{- with .Acc }}<tr><td>acc</td><td>{{ .Unstressed }}</td>{{ template "arb-img" . }}</tr>{{ end -}}
	{{- with .Inst }}<tr><td>inst</td><td>{{ .Unstressed }}</td>{{ template "arb-img" . }}</tr>{{ end -}}
	{{- with .Prep }}<tr><td>prep</td><td>{{ .Unstressed }}</td>{{ template "arb-img" . }}</tr>{{ end -}}
{{- end -}}
{{- end -}}

{{- define "word" -}}
<td class="smol">
{{- if or .AdjInfo .VerbInfo -}}
<a href="{{ absWordInfo . }}">{{- template "wordStr" . -}}</a>
{{- else -}}
{{- template "wordStr" . -}}
{{- end -}}
<div class="scrape">
<a href="{{ absImg . }}">img</a>
<a href="{{ absAudio . }}">audio</a>
</div>
<audio controls>
<source src="{{ absAudio . }}" type="audio/mpeg">
</audio>
</td>
<td class="img-container"><a class="img" href="#"><img src="{{ absImg . }}"/></a></td>
<td class="smol">
{{- if .NounInfo }} {{ template "gender" .NounInfo.Gender }}{{ end -}}
</td>
<td class="smol">
{{- .WordType -}}
</td>
<td class="smol">
{{- if .DerivedFrom }}<a href="{{ absWord .DerivedFrom }}">{{ .DerivedFrom.Word }}</a>{{ end -}}
</td>
<td>
<ul>
{{- range .Translations -}}
<li>{{ template "trans" . }}</li>
{{- end -}}
</ul>
</td>
<td class="smollish">
<a class="c-normal copy" href="#">copy</a>
{{- if ne .Word .Stressed -}}
<a class="c-stressed copy" href="#">copy stressed</a>
{{- end -}}
</td>
{{- end -}}

{{- define "wordStr" -}}
<span class="normal">{{ unstressed . }}</span><span class="stressed">{{ stressednc . }}</span><br/>
{{- end -}}

{{- define "header" -}}
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<title>{{ . }}</title>
	<link rel="shortcut icon" type="image/png" href="/f/fav.png"/>
	<style>
		*                      { padding: 0; margin: 0; box-sizing: border-box; }
		html, body             { background-color: #151515; color: #fff; }
		main                   { max-width: 1400px; width: 95%; margin: 0 auto 50px auto; margin-top: 20px; }
		img.gender             { width: 25px; height: auto; }
		.stressed              { display: none; }
		.copy                  { display: block; transition: color 500ms; }
		.copy.copied           { color: #afa; }
		.copy.error            { color: #faa; }
		.main-table            { width: 100%; }
		.results               { margin-top: 40px; }
		td                     { padding: 20px; width: 20%; }
		td.smol                { width: 5%; }
		td.smollish            { width: 10%; }
		td.img-container       { text-align: center; }
		img                    { height: 150px; width: auto; }
		audio                  { display: none; }
		a                      { color: #ccc; text-decoration: underline; }
		.scrape                { display: none; }
		form                   { position: relative; }
		form input             { min-height: 2em; font-size: 2em; background-color: #333; color: #fff; outline: none; border: 1px solid #ccc; padding: 20px; width: 89%; }
		form .submit           { position: absolute; top: 0; right: 0; width: 10%; margin-left: 1%; }
		.edits                 { font-size: 2em; display: inline-block; width: auto; border: 3px #800 solid; padding: 2px 1em; }
		.edit                  { padding: 5px 0; }
		.edit.h                { display: none; }
		.edit.a                { background-color: #800; color: #800; }
		.edit.d,
		.edit.c                { background-color: #800; color: #fff; }
		.meta                  { margin-left: 20px; }
		.meta table            { margin-top: 2em;  }
		.meta .gender          { width: 16px; }
		.meta td:first-child   {  }
		.meta td               { width: auto; height: 2em; padding: 0 20px 0 0; }
		.meta td.img-container img { max-height: 150%; width: auto; }
		img                    { image-rendering: crisp-edges; }
		}
	</style>
</head>
<body>
<main>
{{- end -}}

{{- define "footer" -}}
</main>
</body>
</html>
{{- end -}}

{{- define "results" -}}
{{- with .Edits -}}
<div class="edits"/>
{{- range . -}}
<span class="edit {{ editType .Type }}">{{ . }}</span>
{{- end -}}
</div>
{{- end -}}
{{- if .Words -}}
<table class="main-table">
{{- range .Words -}}
<tr>{{ template "word" . }}</tr>
{{- end -}}
</table>
{{- else -}}
No results
{{- end -}}
{{- end -}}

{{- define "main" -}}
<div class="input">
<form>
<input type="text"   class="val"    value="{{ .Query }}" placeholder="Слово | Word" />
<input type="submit" class="submit" value=">"                                       />
</form>
</div>
<div class="results">
{{- template "results" . -}}
</div>
{{- end -}}

{{- define "js" -}}<script>`) + data.AppJS + nonl(`</script>{{- end -}}
{{- template "header" "GoRussian" -}}
{{- template "main" . -}}
{{- template "js" -}}
{{- template "footer" }}`)
