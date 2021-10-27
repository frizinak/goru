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

	"github.com/frizinak/goru/common"
	"github.com/frizinak/goru/data"
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
	resultsTpl   *template.Template
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
	case strings.HasPrefix(p, "asset/") && strings.Count(p, "/") == 1:
		return app.handleAsset, 0
	}

	return nil, 0
}

func absWord(w *openrussian.Word) string  { return fmt.Sprintf("/w/%s", w.Word) }
func absImg(w *openrussian.Word) string   { return fmt.Sprintf("/i/%d.png", w.ID) }
func absAudio(w *openrussian.Word) string { return fmt.Sprintf("/a/%d/%s", w.ID, w.Word) }
func absArbitraryAudio(w string) string   { return fmt.Sprintf("/aa//%s", w) }

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

func (app *App) handleAsset(w http.ResponseWriter, r *http.Request, l *log.Logger) (int, error) {
	p := strings.SplitN(r.URL.Path, "/", 2)
	h := w.Header()
	switch p[1] {
	case "n.png":
		h.Set("content-type", "image/png")
		h.Set("cache-control", "max-age=86400")
		w.Write(data.ImgN)
		return 0, nil
	case "f.png":
		h.Set("content-type", "image/png")
		h.Set("cache-control", "max-age=86400")
		w.Write(data.ImgF)
		return 0, nil
	case "m.png":
		h.Set("content-type", "image/png")
		h.Set("cache-control", "max-age=86400")
		w.Write(data.ImgM)
		return 0, nil
	case "fav.png":
		h.Set("content-type", "image/png")
		h.Set("cache-control", "max-age=86400")
		w.Write(data.ImgFav)
		return 0, nil
	}

	return http.StatusNotFound, nil
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

	h := w.Header()
	h.Set("content-type", "audio/mpeg")
	h.Set("cache-control", "max-age=86400")
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

	h := w.Header()
	h.Set("content-type", "image/png")
	h.Set("cache-control", "max-age=86400")
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

	var audio string
	if len(res) != 0 && strings.EqualFold(p[1], res[0].Word) {
		audio = absAudio(res[0])
	}

	reqw := strings.ToLower(r.Header.Get("X-Requested-With"))
	xhr := reqw == "fetch" || reqw == "xmlhttprequest"

	d := WordPage{Query: p[1], Audio: audio, Words: res}

	w.Header().Set("content-type", "text/html")
	if xhr {
		return 0, app.resultsTpl.Execute(w, d)
	}

	return 0, app.wordsTpl.Execute(w, d)
}

type WordPage struct {
	Query string
	Audio string
	Words []*openrussian.Word
}

func main() {
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

	tpl := template.Must(mtpl.Funcs(template.FuncMap{
		"absWord":  absWord,
		"absImg":   absImg,
		"absAudio": absAudio,
		"genderImg": func(gender openrussian.Gender) string {
			switch gender {
			case openrussian.N:
				return "/asset/n.png"
			case openrussian.F:
				return "/asset/f.png"
			case openrussian.M:
				return "/asset/m.png"
			default:
				return ""
			}
		},
	}).Parse(
		`{{- define "trans" -}}
<div>{{ .Translation }}</div>
{{ if .Info }}<div>{{ .Info }}</div>{{ end }}
{{- if .Example -}}
<div>
	<p>{{ .Example -}}</p>
	{{ if .ExampleTranslation }}<p>{{ .ExampleTranslation}}</p>{{ end }}
</div>
{{- end -}}
{{- end -}}

{{- define "gender" -}}{{ with genderImg . }}<img src="{{ . }}"/>{{ end }}{{- end -}}

{{- define "word" -}}
<td class="smol">
{{- template "wordStr" . -}}
<div class="scrape">
<a href="{{ absImg . }}">img</a>
<a href="{{ absAudio . }}">audio</a>
</div>
<audio controls>
<source src="{{ absAudio . }}" type="audio/mpeg">
</audio>
</td>
<td class="img-container"><a class="img" href="#"><img src="{{ absImg . }}"/></a></td>
<td class="smol gender">
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
{{- end }}
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
	<link rel="shortcut icon" type="image/png" href="/asset/fav.png"/>
	<style>
		*                { padding: 0; margin: 0; box-sizing: border-box; }
		html, body       { background-color: #151515; color: #fff; }
		main             { max-width: 1400px; width: 95%; margin: 0 auto 0 auto; margin-top: 20px; }
		.gender img      { width: 25px; height: auto; }
		.stressed        { display: none; }
		.copy            { display: block; transition: color 500ms; }
		.copy.copied     { color: #afa; }
		.copy.error      { color: #faa; }
		.results table   { width: 100%; }
		.results         { margin-top: 40px; }
		td               { padding: 20px; width: 20%; }
		td.smol          { width: 5%; }
		td.smollish      { width: 10%; }
		td.img-container { text-align: center; }
		img              { height: 150px; width: auto; }
		audio            { display: none; }
		a                { color: #ccc; text-decoration: underline; }
		.scrape          { display: none; }
		form             { position: relative; }
		form input       { min-height: 2em; font-size: 2em; background-color: #333; color: #fff; outline: none; border: 1px solid #ccc; padding: 20px; width: 89%; }
		form .submit     { position: absolute; top: 0; right: 0; width: 10%; margin-left: 1%; }
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
{{ if .Words -}}
<table>
{{- range .Words }}
<tr>{{ template "word" . }}</tr>
{{ end -}}
</table>
{{ else -}}
No results
{{ end -}}
{{- end -}}

{{ template "header" "GoRussian" }}
<div class="input">
<form>
<input type="text"   class="val"    value="{{ .Query }}" />
<input type="submit" class="submit" value=">"            />
</form>
</div>
<div class="results">
{{ template "results" . }}
</div>
<script>` + data.AppJS + `</script>
{{ template "footer" }}`,
	))

	homeTpl := template.Must(tpl.New("home").Parse(`
{{- template "header" . }}
<a href="/w/Здравствуйте"><h1>Здравствуйте</h1></a>
{{ template "footer" }}`))

	scrapableTpl := template.Must(template.Must(tpl.Clone()).Parse(`
{{- define "word" -}}
<a href="{{ absImg . }}">w</a>
<a href="{{ absAudio . }}">w</a>
{{- end -}}
`))

	errTpl := template.Must(tpl.New("err").Parse(`
{{- template "header" "Error" }}
	{{ . }}
{{ template "footer" }}`))

	resultsTpl := template.Must(tpl.New("xhr").Parse(`
{{- template "results" . -}}
`))

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
		resultsTpl:   resultsTpl,
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
