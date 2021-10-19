package main

import (
	"flag"
	"fmt"
	"html/template"
	"image/color"
	"image/png"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "net/http/pprof"

	"github.com/frizinak/goru/common"
	"github.com/frizinak/goru/image"
	"github.com/frizinak/goru/openrussian"
	"github.com/frizinak/gotls/simplehttp"
	"github.com/frizinak/gotls/tls"
)

type App struct {
	tpl *template.Template
}

func (app *App) route(r *http.Request, l *log.Logger) (simplehttp.HandleFunc, int) {
	p := strings.Trim(r.URL.Path, "/")
	r.URL.Path = p

	switch p {
	case "":
		return app.handleHome, 0
	}

	switch {
	case strings.HasPrefix(p, "w/") && strings.Count(p, "/") == 1:
		return app.handleWord, 0
	case strings.HasPrefix(p, "img/") && strings.Count(p, "/") == 1:
		return app.handleImg, 0
	}

	return nil, 0
}

func (app *App) handleHome(w http.ResponseWriter, r *http.Request, l *log.Logger) (int, error) {
	fmt.Fprint(
		w,
		`<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF8">
	<title>GoRussian</title>
</head>
<body><a href="/w/Здравствуйте"><h1>Здравствуйте</h1></a></body>
</html>`,
	)
	return 0, nil
}

var (
	imgFG = color.NRGBA{0, 0, 0, 255}
	imgBG = color.NRGBA{0, 0, 0, 0}
)

func (app *App) handleImg(w http.ResponseWriter, r *http.Request, l *log.Logger) (int, error) {
	p := strings.SplitN(r.URL.Path, "/", 2)
	if !strings.HasSuffix(p[1], ".png") {
		return http.StatusNotFound, nil
	}
	n, _ := strconv.Atoi(p[1][:len(p[1])-4])
	if n <= 0 {
		return http.StatusNotFound, nil
	}

	dict, err := common.GetDict()
	if err != nil {
		return 0, err
	}
	word := dict.Words()[openrussian.ID(n)]
	if word == nil {
		return http.StatusNotFound, nil
	}
	w.Header().Set("content-type", "image/png")
	img, err := image.Image(300, word.Stressed.Parse().String(), imgFG, imgBG)
	if err != nil {
		return 0, err
	}

	//return 0, jpeg.Encode(w, img, &jpeg.Options{Quality: 80})
	return 0, png.Encode(w, img)
}

func (app *App) handleWord(w http.ResponseWriter, r *http.Request, l *log.Logger) (int, error) {
	p := strings.SplitN(r.URL.Path, "/", 2)
	dict, err := common.GetDict()
	if err != nil {
		return 0, err
	}

	res, cyr := dict.Search(p[1], true, 30)
	if len(res) == 0 && cyr {
		res = dict.SearchRussianFuzzy(p[1], true, 30)
	}
	if len(res) == 0 {
		return http.StatusNotFound, nil
	}

	w.Header().Set("content-type", "text/html")
	return 0, app.tpl.Execute(w, res)
}

func main() {
	go func() {
		log.Fatal(http.ListenAndServe(":6060", nil))
	}()
	var addr string
	flag.StringVar(&addr, "l", ":80", "address to bind to")
	flag.Parse()

	mtpl, err := common.GetHTMLTpl()
	if err != nil {
		panic(err)
	}

	tpl, err := mtpl.Parse(
		`{{- define "trans" -}}
<div>{{ .Translation }}</div>
{{ if .Info }}<div>{{.Info }}</div>{{ end }}
{{ if .Example -}}
<div>
	<p>{{ .Example -}}</p>
	{{ if .ExampleTranslation }}<p>{{ .ExampleTranslation}}</p>{{ end }}
</div>
{{ end -}}
{{- end -}}

{{- define "gender" -}}{{ genderSymbol . }}{{- end -}}

{{- define "word" -}}
<td>
{{- template "wordStr" . -}}
</td>
<td>
<div class="img"><img src="/img/{{ .ID }}.png"/></div>
</td>
<td>
{{- if .NounInfo }} {{ template "gender" .NounInfo.Gender }}{{ end -}}
</td>
<td>
{{- .WordType -}}
</td>
<td>
{{- if .DerivedFrom }}<a href="/w/{{ .DerivedFrom.Word }}">{{ .DerivedFrom.Word }}</a>{{ end -}}
</td>
<td>
<ul>
{{- range .Translations }}
<li>{{ template "trans" . }}</li>
{{ end -}}
</ul>
</td>
<td>
<audio controls>
  <source src="https://api.openrussian.org/read/ru/{{ .Word }}" type="audio/mpeg">
</audio>
</td>
{{- end -}}

{{- define "wordStr" -}}
{{- stressednc . -}}
{{- end -}}


<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF8">
	<title>GoRussian</title>
	<style>
		* { padding: 0; margin: 0; }
		table { max-width: 1400px; width: 95%; margin: 0 auto 0 auto; }
		td { padding: 20px; text-align: center; }
		.img { min-width: 600px; }
		img { height: 150px; width: auto; }
	</style>
</head>
<body>
	<table>
		{{- range . }}
		<tr>
{{ template "word" . }}
		</tr>
		{{ end -}}
	</table>

	<script>
		let els = document.getElementsByClassName('img');
		let audios = document.getElementsByTagName('audio');
		for (let i = 0; i < audios.length; i++) {
			audios[i].style.display = 'none';
		}
		for (let i = 0; i < els.length; i++) {
			els[i].onclick = function(e) {
				return function() {
					let audio = e.parentElement.parentElement.getElementsByTagName('audio')[0];
					let wasP = audio.paused || audio.ended;
					for (let i = 0; i < audios.length; i++) {
						audios[i].pause();
						audios[i].currentTime = 0;
					}
					if (wasP) {
						audio.volume = 1;
						audio.play();
					}
				};
			}(els[i])
		}
	</script>
</body>
</html>`,
	)
	if err != nil {
		panic(err)
	}

	l := log.New(os.Stderr, "", log.Ldate|log.Ltime)
	app := &App{tpl}
	s := tls.New(app.route, l)
	l.Fatal(s.Start(addr, false))
}
