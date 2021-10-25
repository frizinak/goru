package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"image/color"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "net/http/pprof"

	"github.com/frizinak/goru/common"
	"github.com/frizinak/goru/image"
	"github.com/frizinak/goru/openrussian"
	"github.com/frizinak/gotls/simplehttp"
	"github.com/frizinak/gotls/tls"
)

var (
	imgFG = color.NRGBA{255, 255, 255, 255}
	imgBG = color.NRGBA{0, 0, 0, 0}
)

type Config struct {
	AudioCacheDir string
	ImageCacheDir string
}

type App struct {
	conf         Config
	homeTpl      *template.Template
	wordsTpl     *template.Template
	scrapableTpl *template.Template
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
	case strings.HasPrefix(p, "a/") && strings.Count(p, "/") == 2:
		return app.handleAudio, 0
	case strings.HasPrefix(p, "i/") && strings.Count(p, "/") == 1:
		return app.handleImg, 0
	}

	return nil, 0
}

func (app *App) cache(path string, w io.Writer, generate func(w io.Writer) (int64, error)) (int64, error) {
	f, err := os.Open(path)
	if err == nil {
		n, err := io.Copy(w, f)
		f.Close()
		return n, err
	}

	if os.IsNotExist(err) {
		tmp := fmt.Sprintf("%s.%d.tmp", path, time.Now().UnixNano())
		f, err := os.Create(tmp)
		if err != nil {
			return 0, err
		}
		rw := io.MultiWriter(f, w)
		n, err := generate(rw)
		f.Close()
		if err != nil {
			os.Remove(tmp)
			return n, err
		}
		os.Rename(tmp, path)
		return n, nil
	}

	return 0, err
}

func (app *App) img(word *openrussian.Word, w io.Writer) (int64, error) {
	if word == nil {
		return 0, errors.New("nil word")
	}

	fp := filepath.Join(app.conf.ImageCacheDir, strconv.Itoa(int(word.ID)))
	return app.cache(fp, w, func(w io.Writer) (int64, error) {
		img, err := image.Image(300, word.Stressed.Parse().String(), imgFG, imgBG)
		if err != nil {
			return 0, err
		}

		return -1, png.Encode(w, img)
	})
}

func (app *App) audio(word *openrussian.Word, w io.Writer) (int64, error) {
	if word == nil {
		return 0, errors.New("nil word")
	}

	fp := filepath.Join(app.conf.AudioCacheDir, strconv.Itoa(int(word.ID)))
	return app.cache(fp, w, func(w io.Writer) (int64, error) {
		uri := fmt.Sprintf("https://api.openrussian.org/read/ru/%s", word.Word)
		res, err := http.Get(uri)
		if err != nil {
			return 0, err
		}
		defer res.Body.Close()
		return io.Copy(w, res.Body)
	})
}

func (app *App) handleAudio(w http.ResponseWriter, r *http.Request, l *log.Logger) (int, error) {
	p := strings.SplitN(r.URL.Path, "/", 3)
	iID, err := strconv.Atoi(p[1])
	if err != nil {
		return http.StatusNotFound, nil
	}
	iWord := p[2]

	dict, err := common.GetDict()
	if err != nil {
		return 0, err
	}
	all := dict.Words()
	word := all[openrussian.ID(iID)]
	if word == nil || word.Word != iWord {
		return http.StatusNotFound, nil
	}

	w.Header().Set("content-type", "audio/mpeg")
	_, err = app.audio(word, w)

	return 0, err
}

func (app *App) handleHome(w http.ResponseWriter, r *http.Request, l *log.Logger) (int, error) {
	w.Header().Set("content-type", "text/html")
	return 0, app.homeTpl.Execute(w, "GoRussian")
	// dict, err := common.GetDict()
	// if err != nil {
	// 	return 0, err
	// }

	// w.Header().Set("content-type", "text/html")
	// return 0, app.scrapableTpl.Execute(w, dict.Words())
}

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
	_, err = app.img(word, w)

	return 0, err
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
	return 0, app.wordsTpl.Execute(w, res)
}

func main() {
	go func() {
		log.Fatal(http.ListenAndServe(":6060", nil))
	}()
	var addr string
	var cacheDir string
	flag.StringVar(&addr, "l", ":80", "address to bind to")
	flag.StringVar(&cacheDir, "c", "", "cache dir, defaults to <XDG default>/goru")
	flag.Parse()

	if cacheDir == "" {
		_cacheDir, err := os.UserCacheDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "please specify a cache dir (-c) as we could not find a default directory: %s\n", err)
			os.Exit(1)
		}
		cacheDir = filepath.Join(_cacheDir, "goru")
	}

	mtpl, err := common.GetHTMLTpl()
	if err != nil {
		panic(err)
	}

	tpl := template.Must(mtpl.Parse(
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
<div class="scrape">
<a href="/i/{{ .ID }}.png">img</a>
<a href="/a/{{ .ID }}/{{ .Word }}">audio</a>
</div>
</td>
<td class="img-container">
<a class="img"><img src="/i/{{ .ID }}.png"/></a>
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
  <source src="/a/{{ .ID }}/{{ .Word }}" type="audio/mpeg">
</audio>
</td>
{{- end -}}

{{- define "wordStr" -}}
{{- stressednc . -}}
{{- end -}}

{{- define "header" -}}
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF8">
	<title>{{ . }}</title>
	<style>
		* { padding: 0; margin: 0; }
		html, body { background-color: #151515; color: #fff; }
		main { max-width: 1400px; width: 95%; margin: 0 auto 0 auto; margin-top: 20px; }
		td { padding: 20px; }
		td.img-container { text-align: center; }
		.img { display: block; min-width: 600px; }
		img { height: 150px; width: auto; }
		audio { display: none; }
		a { color: #ccc; text-decoration: underline; }
		.scrape { display: none; }
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

{{ template "header" "GoRussian" }}
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
{{ template "footer" }}`,
	))

	homeTpl := template.Must(tpl.New("home").Parse(`
{{- template "header" . }}
<a href="/w/Здравствуйте"><h1>Здравствуйте</h1></a>
{{ template "footer" }}`))

	scrapableTpl := template.Must(template.Must(tpl.Clone()).Parse(`
{{- define "word" -}}
<a href="/i/{{ .ID }}.png">w</a>
<a href="/a/{{ .ID }}/{{ .Word }}">w</a>
{{- end -}}
`))

	errTpl := template.Must(tpl.New("err").Parse(`
{{- template "header" "Error" }}
	{{ . }}
{{ template "footer" }}`))

	audioCacheDir := filepath.Join(cacheDir, "audio")
	imgCacheDir := filepath.Join(cacheDir, "img")
	os.MkdirAll(audioCacheDir, 0700)
	os.MkdirAll(imgCacheDir, 0700)

	l := log.New(os.Stderr, "", log.Ldate|log.Ltime)
	app := &App{
		conf: Config{
			AudioCacheDir: audioCacheDir,
			ImageCacheDir: imgCacheDir,
		},
		wordsTpl:     tpl,
		homeTpl:      homeTpl,
		scrapableTpl: scrapableTpl,
	}
	s := tls.New(app.route, l)

	buf := bytes.NewBuffer(nil)
	for i := 300; i <= 500; i++ {
		buf.Reset()
		errstr := http.StatusText(i)
		if errstr == "" {
			errstr = "Something went wrong"
		}
		if err := errTpl.Execute(buf, fmt.Sprintf("%d - %s", i, errstr)); err != nil {
			panic(err)
		}
		b := make([]byte, buf.Len())
		copy(b, buf.Bytes())
		s.SetHTTPErrorHandler(i, simplehttp.NewHTTPError("text/html", b))
	}

	l.Fatal(s.Start(addr, false))
}
