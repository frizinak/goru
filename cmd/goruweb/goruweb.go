package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
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
	"github.com/frizinak/goru/dict"
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
	prod         bool
	rate         chan struct{}
	conf         Config
	homeTpl      *template.Template
	wordsTpl     *template.Template
	wordTpl      *template.Template
	resultsTpl   *template.Template
	scrapableTpl *template.Template
}

func (app *App) ratelimit(h simplehttp.HandleFunc) simplehttp.HandleFunc {
	return func(w http.ResponseWriter, r *http.Request, l *log.Logger) (int, error) {
		app.rate <- struct{}{}
		n, err := h(w, r, l)
		<-app.rate
		return n, err
	}
}

func (app *App) route(r *http.Request, l *log.Logger) (simplehttp.HandleFunc, int) {
	if app.prod && !strings.HasPrefix(r.RemoteAddr, "192.168.") {
		buf := bytes.NewBuffer(make([]byte, 0, 255))
		buf.WriteString("CONN[")
		buf.WriteString(r.RemoteAddr)
		buf.WriteString("] UA[")
		buf.WriteString(r.UserAgent())
		buf.WriteString("] PATH[")
		buf.WriteString(r.Method)
		buf.WriteString(" ")
		buf.WriteString(r.URL.String())
		buf.WriteString("]")
		ref := r.Referer()
		if strings.Contains(ref, "home.friz.pro") {
			ref = ""
		}
		buf.WriteString(" REF[")
		buf.WriteString(ref)
		buf.WriteString("]")
		buf.WriteByte(10)
		buf.WriteTo(os.Stdout)
	}
	p := strings.Trim(r.URL.Path, "/")
	r.URL.Path = p

	switch p {
	case "":
		return app.handleHome, 0
	}

	switch {
	case strings.HasPrefix(p, "w/") && strings.Count(p, "/") == 1:
		return app.ratelimit(app.handleWord), 0
	case strings.HasPrefix(p, "w/i/") && strings.Count(p, "/") == 2:
		return app.handleWordInfo, 0
	case strings.HasPrefix(p, "a/") && strings.Count(p, "/") == 2:
		return app.handleAudio, 0
	case strings.HasPrefix(p, "aa/") && strings.Count(p, "/") == 2:
		return app.handleArbitaryAudio, 0
	case strings.HasPrefix(p, "i/") && strings.Count(p, "/") == 1:
		return app.handleImg, 0
	case strings.HasPrefix(p, "ai/") && strings.Count(p, "/") == 2:
		return app.handleArbitaryImg, 0
	case strings.HasPrefix(p, "f/") && strings.Count(p, "/") == 1:
		return app.handleAsset, 0
	}

	return nil, 0
}

var b64e = base64.NewEncoding(Base64Chars).WithPadding(base64.NoPadding)

func absWord(w *openrussian.Word) string     { return fmt.Sprintf("/w/%s", w.Word) }
func absWordInfo(w *openrussian.Word) string { return fmt.Sprintf("/w/i/%d", w.ID) }
func absImg(w *openrussian.Word) string      { return fmt.Sprintf("/i/%d.png", w.ID) }
func absAudio(w *openrussian.Word) string    { return fmt.Sprintf("/a/%d/%s", w.ID, w.Word) }

func sign(data string) string {
	s1 := sha256.New()
	s1.Write([]byte(data))
	s1.Write(URLSigSalt)
	buf := s1.Sum(make([]byte, 0, 64))
	s2 := sha256.New()
	s2.Write(buf)
	buf = s2.Sum(buf)
	return b64e.EncodeToString(buf[32:])
}

func absArbitraryAudio(w string) string { return fmt.Sprintf("/aa/%s/%s", sign(w), w) }
func absArbitraryImg(w string) string   { return fmt.Sprintf("/ai/%s/%s.png", sign(w), w) }

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
		str := word.Stressed.Parse().String()
		img, err := image.Image(300, str, str, false, imgFG, imgBG)
		if err != nil {
			return 0, err
		}

		return -1, png.Encode(w, img)
	})
}

func (app *App) arbimg(word string, w io.Writer) (int64, error) {
	if len(word) == 0 {
		return 0, errors.New("nil word")
	}

	fn := fmt.Sprintf("ai-%s", sign(word))
	fp := filepath.Join(app.conf.ImageCacheDir, fn)
	return app.cache(fp, w, func(w io.Writer) (int64, error) {
		img, err := image.Image(150, "", word, false, imgFG, imgBG)
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

func (app *App) arbaudio(word string, w io.Writer) (int64, error) {
	if len(word) == 0 {
		return 0, errors.New("nil word")
	}

	fn := fmt.Sprintf("aa-%s", sign(word))
	fp := filepath.Join(app.conf.AudioCacheDir, fn)
	return app.cache(fp, w, func(w io.Writer) (int64, error) {
		uri := fmt.Sprintf("https://api.openrussian.org/read/ru/%s", word)
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

func (app *App) handleArbitaryAudio(w http.ResponseWriter, r *http.Request, l *log.Logger) (int, error) {
	p := strings.SplitN(r.URL.Path, "/", 3)
	s := p[1]
	rawstr := p[2]
	if sign(rawstr) != s {
		return http.StatusNotAcceptable, nil
	}
	h := w.Header()
	h.Set("content-type", "audio/mpeg")
	h.Set("cache-control", "max-age=86400")
	_, err := app.arbaudio(rawstr, w)

	return 0, err
}

func (app *App) handleHome(w http.ResponseWriter, r *http.Request, l *log.Logger) (int, error) {
	dict, err := common.GetDict()
	if err != nil {
		return 0, err
	}
	words := dict.Words()
	d := WordPage{Query: "", Words: []*openrussian.Word{words[33002]}}

	w.Header().Set("content-type", "text/html")
	return 0, app.wordsTpl.Execute(w, d)

	// w.Header().Set("content-type", "text/html")
	// return 0, app.homeTpl.Execute(w, "GoRussian")

	// dict, err := common.GetDict()
	// if err != nil {
	// 	return 0, err
	// }

	// w.Header().Set("content-type", "text/html")
	// mp := dict.Words()
	// words := make([]*openrussian.Word, 0, len(mp))
	// for _, w := range mp {
	// 	words = append(words, w)
	// }
	// return 0, app.scrapableTpl.Execute(w, WordPage{Words: words})
}

func (app *App) handleArbitaryImg(w http.ResponseWriter, r *http.Request, l *log.Logger) (int, error) {
	p := strings.SplitN(r.URL.Path, "/", 3)
	if !strings.HasSuffix(p[2], ".png") {
		return http.StatusNotFound, nil
	}
	s := p[1]
	rawstr := p[2][:len(p[2])-4]
	if sign(rawstr) != s {
		return http.StatusNotAcceptable, nil
	}
	str := openrussian.SplitStressed(rawstr)
	h := w.Header()
	h.Set("content-type", "image/png")
	h.Set("cache-control", "max-age=86400")
	_, err := app.arbimg(str.String(), w)

	return 0, err
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
	dct, err := common.GetDict()
	if err != nil {
		return 0, err
	}

	const max = 30
	var res []*openrussian.Word
	res, cyr := dct.SearchFuzzy(p[1], true, max)

	var audio string
	if len(res) != 0 && strings.EqualFold(p[1], res[0].Word) {
		audio = absAudio(res[0])
	}

	reqw := strings.ToLower(r.Header.Get("X-Requested-With"))
	xhr := reqw == "fetch" || reqw == "xmlhttprequest"

	var edits dict.Edits
	if cyr && len(res) != 0 {
		q := []rune(p[1])
		edits = dict.LevenshteinEdits([]rune(res[0].Word), q)
		if !edits.HasEdits() {
			edits = nil
		}
	}

	d := WordPage{Query: p[1], Edits: edits, Audio: audio, Words: res}
	w.Header().Set("content-type", "text/html")
	if xhr {
		return 0, app.resultsTpl.Execute(w, d)
	}

	return 0, app.wordsTpl.Execute(w, d)
}

func (app *App) handleWordInfo(w http.ResponseWriter, r *http.Request, l *log.Logger) (int, error) {
	p := strings.SplitN(r.URL.Path, "/", 3)
	id, err := strconv.Atoi(p[2])
	if err != nil {
		return http.StatusNotFound, nil
	}

	dct, err := common.GetDict()
	if err != nil {
		return 0, err
	}
	word := dct.Words()[openrussian.ID(id)]
	if word == nil {
		return http.StatusNotFound, nil
	}

	d := WordPage{Query: word.Word, Edits: nil, Audio: "", Words: []*openrussian.Word{word}}
	w.Header().Set("content-type", "text/html")
	return 0, app.wordTpl.Execute(w, d)
}

type WordPage struct {
	Query string
	Edits dict.Edits
	Audio string
	Words []*openrussian.Word
}

func main() {
	var addr string
	var cacheDir string
	if !Prod {
		flag.StringVar(&addr, "l", ":8080", "address to bind to")
	}

	flag.StringVar(&cacheDir, "c", "", "cache dir, defaults to <XDG default>/goru")
	flag.Parse()

	l := log.New(os.Stderr, "", log.Ltime|log.Lmicroseconds)
	l.Println("initializing")
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

	nonl := func(i string) string { return strings.ReplaceAll(strings.ReplaceAll(i, "\n", ""), "\t", "") }

	tpl := template.Must(mtpl.Funcs(template.FuncMap{
		"editType": func(t dict.EditType) string {
			switch t {
			case dict.EditNone:
				return "k"
			case dict.EditAdd:
				return "a"
			case dict.EditDel:
				return "d"
			case dict.EditChange:
				return "c"
			}
			return "h"
		},
		"absArbitraryImg":   absArbitraryImg,
		"absArbitraryAudio": absArbitraryAudio,
		"absWord":           absWord,
		"absWordInfo":       absWordInfo,
		"absImg":            absImg,
		"absAudio":          absAudio,
		"genderImg": func(gender interface{}) string {
			if g, ok := gender.(openrussian.Gender); ok {
				switch g {
				case openrussian.N:
					return "/f/n.png"
				case openrussian.F:
					return "/f/f.png"
				case openrussian.M:
					return "/f/m.png"
				case openrussian.Pl:
					return "/f/pl.png"
				default:
					return ""
				}
			}

			return fmt.Sprintf("/f/%s.png", gender.(string))
		},
	}).Parse(nonl(`{{- define "trans" -}}
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
<a class="img" href="#"><img src="{{ absArbitraryImg .String }}"/></a>
<audio controls>
<source src="{{ absArbitraryAudio .Unstressed }}" type="audio/mpeg">
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
{{- template "footer" }}`),
	))

	homeTpl := template.Must(tpl.New("home").Parse(`
{{- template "header" . -}}
<a href="/w/Здравствуйте"><h1>Здравствуйте</h1></a>
{{- template "footer" }}`))

	scrapableTpl := template.Must(template.Must(tpl.Clone()).Parse(`
{{- define "word" -}}
<a href="{{ absWord . }}">w</a>
{{- end -}}`))

	errTpl := template.Must(tpl.New("err").Parse(`
{{- template "header" "Error" -}}
	{{ . -}}
{{- template "footer" }}`))

	resultsTpl := template.Must(tpl.New("xhr").Parse(`
{{- template "results" . -}}`))

	wordInfoTpl := template.Must(tpl.New("info").Parse(`
{{- template "header" (index .Words 0) -}}
{{- template "main" . -}}
{{- template "word-info" (index .Words 0) -}}
{{- template "js" . -}}
{{- template "footer" }}`))

	audioCacheDir := filepath.Join(cacheDir, "audio")
	imgCacheDir := filepath.Join(cacheDir, "img")
	os.MkdirAll(audioCacheDir, 0700)
	os.MkdirAll(imgCacheDir, 0700)

	app := &App{
		prod: Prod,
		rate: make(chan struct{}, 3),
		conf: Config{
			AudioCacheDir: audioCacheDir,
			ImageCacheDir: imgCacheDir,
		},
		wordsTpl:     tpl,
		wordTpl:      wordInfoTpl,
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

	d, err := common.GetDict()
	if err != nil {
		l.Fatal(err)
	}
	l.Println("loaded dictionary")
	d.InitEnglishFuzzIndex()
	l.Println("initialized english index")
	d.InitRussianFuzzIndex()
	l.Println("initialized russian index")
	l.Fatal(run(s, addr))
}
