package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"image/color"
	"image/png"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	AudioCacheDir          string
	ImageCacheDir          string
	ArbitraryAudioCacheDir string
	ArbitraryImageCacheDir string
}

type App struct {
	prod         bool
	cpurate      chan struct{}
	netrate      chan struct{}
	conf         Config
	homeTpl      *template.Template
	wordsTpl     *template.Template
	wordTpl      *template.Template
	resultsTpl   *template.Template
	scrapableTpl *template.Template

	scrape struct {
		l     sync.Mutex
		words []*openrussian.Word
	}
}

func (app *App) ratelimit(h simplehttp.HandleFunc) simplehttp.HandleFunc {
	if app.cpurate == nil {
		return h
	}

	return func(w http.ResponseWriter, r *http.Request, l *log.Logger) (int, error) {
		app.cpurate <- struct{}{}
		n, err := h(w, r, l)
		<-app.cpurate
		return n, err
	}
}

func slash(r rune) bool { return r == '/' }

type uri struct {
	raw     string
	correct string
	parts   []string
}

func parse(u *url.URL) (*uri, error) {
	raw := u.EscapedPath()
	parts := strings.FieldsFunc(strings.TrimFunc(raw, slash), slash)
	correct := "/" + strings.Join(parts, "/")
	var err error
	for i, v := range parts {
		parts[i], err = url.PathUnescape(v)
		if err != nil {
			return nil, err
		}
	}

	return &uri{raw: raw, correct: correct, parts: parts}, nil
}

type handler func(w http.ResponseWriter, r *http.Request, args []string) (int, error)

func (app *App) wrapArgs(h handler, args []string) simplehttp.HandleFunc {
	return func(w http.ResponseWriter, r *http.Request, l *log.Logger) (int, error) {
		return h(w, r, args)
	}
}

func (app *App) route(r *http.Request, l *log.Logger) (simplehttp.HandleFunc, int) {
	u, err := parse(r.URL)
	if err != nil {
		return nil, http.StatusNotAcceptable
	}
	if u.raw != u.correct {
		return func(w http.ResponseWriter, r *http.Request, l *log.Logger) (int, error) {
			w.Header().Set("Location", u.correct)
			return http.StatusMovedPermanently, nil
		}, 0
	}

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

	switch {
	case len(u.parts) == 0:
		return app.wrapArgs(app.handleHome, u.parts), 0

	case len(u.parts) == 2 && u.parts[0] == "scrape":
		return app.wrapArgs(app.handleScrape, u.parts), 0

	case len(u.parts) == 2 && u.parts[0] == "w":
		return app.ratelimit(app.wrapArgs(app.handleWord, u.parts)), 0

	case len(u.parts) == 3 && u.parts[0] == "w" && u.parts[1] == "i":
		return app.wrapArgs(app.handleWordInfo, u.parts), 0

	case len(u.parts) == 3 && u.parts[0] == "a":
		return app.wrapArgs(app.handleAudio, u.parts), 0

	case len(u.parts) == 3 && u.parts[0] == "aa":
		return app.wrapArgs(app.handleArbitaryAudio, u.parts), 0

	case len(u.parts) == 2 && u.parts[0] == "i":
		return app.wrapArgs(app.handleImg, u.parts), 0

	case len(u.parts) == 3 && u.parts[0] == "ai":
		return app.wrapArgs(app.handleArbitaryImg, u.parts), 0

	case len(u.parts) == 2 && u.parts[0] == "f":
		return app.wrapArgs(app.handleAsset, u.parts), 0
	}

	return nil, 0
}

var b64e = base64.NewEncoding(Base64Chars).WithPadding(base64.NoPadding)

func esc(i string) string                    { return url.PathEscape(i) }
func absWord(w *openrussian.Word) string     { return fmt.Sprintf("/w/%s", esc(w.Word)) }
func absWordInfo(w *openrussian.Word) string { return fmt.Sprintf("/w/i/%d", w.ID) }
func absImg(w *openrussian.Word) string      { return fmt.Sprintf("/i/%d.png", w.ID) }
func absAudio(w *openrussian.Word) string    { return fmt.Sprintf("/a/%d/%s", w.ID, esc(w.Word)) }

func fnhash(data string) string {
	s1 := sha256.New()
	s1.Write([]byte(data))
	return hex.EncodeToString(s1.Sum(nil))
}

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

func absArbitraryAudio(w string) string { return fmt.Sprintf("/aa/%s/%s", sign(w), esc(w)) }
func absArbitraryImg(w string) string   { return fmt.Sprintf("/ai/%s/%s.png", sign(w), esc(w)) }

func (app *App) cache(dir, name string, w io.Writer, rate chan struct{}, generate func(w io.Writer) (int64, error)) (int64, error) {
	hn := fnhash(name)
	fulldir := filepath.Join(dir, hn[0:2], hn[2:4])
	path := filepath.Join(fulldir, hn[4:])
	f, err := os.Open(path)
	if err == nil {
		n, err := io.Copy(w, f)
		f.Close()
		return n, err
	}

	if os.IsNotExist(err) {
		if rate != nil {
			rate <- struct{}{}
			defer func() { <-rate }()
		}
		if _, err := os.Stat(fulldir); err != nil {
			os.MkdirAll(fulldir, 0700)
		}
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

	return app.cache(app.conf.ImageCacheDir, strconv.Itoa(int(word.ID)), w, app.cpurate, func(w io.Writer) (int64, error) {
		str := word.Stressed.Parse().String()
		img, err := image.Image(150, str, str, false, imgFG, imgBG)
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

	return app.cache(app.conf.ArbitraryImageCacheDir, word, w, app.cpurate, func(w io.Writer) (int64, error) {
		img, err := image.Image(40, "", word, false, imgFG, imgBG)
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

	return app.cache(app.conf.AudioCacheDir, strconv.Itoa(int(word.ID)), w, app.netrate, func(w io.Writer) (int64, error) {
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

	return app.cache(app.conf.ArbitraryAudioCacheDir, word, w, app.netrate, func(w io.Writer) (int64, error) {
		uri := fmt.Sprintf("https://api.openrussian.org/read/ru/%s", word)
		res, err := http.Get(uri)
		if err != nil {
			return 0, err
		}
		defer res.Body.Close()
		return io.Copy(w, res.Body)
	})
}

func (app *App) initScrapable() error {
	if app.scrape.words != nil {
		return nil
	}
	app.scrape.l.Lock()
	defer app.scrape.l.Unlock()
	if app.scrape.words != nil {
		return nil
	}
	dict, err := common.GetDict()
	if err != nil {
		return err
	}
	mp := dict.Words()
	l := make([]*openrussian.Word, 0, len(mp))
	for _, w := range mp {
		l = append(l, w)
	}
	app.scrape.words = l
	return nil
}

func (app *App) handleHome(w http.ResponseWriter, r *http.Request, p []string) (int, error) {
	dict, err := common.GetDict()
	if err != nil {
		return 0, err
	}
	words := dict.Words()
	d := WordPage{Query: "", Words: []*openrussian.Word{words[33002]}}

	w.Header().Set("content-type", "text/html")
	return 0, app.wordsTpl.Execute(w, d)
}

func (app *App) handleScrape(w http.ResponseWriter, r *http.Request, p []string) (int, error) {
	page, err := strconv.Atoi(p[1])
	if err != nil || page < 0 {
		return http.StatusNotFound, nil
	}

	if err := app.initScrapable(); err != nil {
		return 0, err
	}
	const max = 50
	offset := max * page
	amount := max
	if offset >= len(app.scrape.words) {
		return http.StatusNotFound, nil
	}

	next := fmt.Sprintf("/scrape/%d", page+1)
	if offset+amount >= len(app.scrape.words) {
		amount = len(app.scrape.words) - offset
		next = ""
	}

	words := app.scrape.words[offset : offset+amount]
	w.Header().Set("content-type", "text/html")
	return 0, app.scrapableTpl.Execute(w, WordPage{Words: words, Next: next})
}

func (app *App) handleAsset(w http.ResponseWriter, r *http.Request, p []string) (int, error) {
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
	case "pl.png":
		h.Set("content-type", "image/png")
		h.Set("cache-control", "max-age=86400")
		w.Write(data.ImgPl)
		return 0, nil
	case "fav.png":
		h.Set("content-type", "image/png")
		h.Set("cache-control", "max-age=86400")
		w.Write(data.ImgFav)
		return 0, nil
	}

	return http.StatusNotFound, nil
}

func (app *App) handleAudio(w http.ResponseWriter, r *http.Request, p []string) (int, error) {
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

func (app *App) handleArbitaryAudio(w http.ResponseWriter, r *http.Request, p []string) (int, error) {
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

func (app *App) handleArbitaryImg(w http.ResponseWriter, r *http.Request, p []string) (int, error) {
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

func (app *App) handleImg(w http.ResponseWriter, r *http.Request, p []string) (int, error) {
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

func (app *App) handleWord(w http.ResponseWriter, r *http.Request, p []string) (int, error) {
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

func (app *App) handleWordInfo(w http.ResponseWriter, r *http.Request, p []string) (int, error) {
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
	Next  string
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
	}).Parse(mainTpl))

	homeTpl := template.Must(tpl.New("home").Parse(`
{{- template "header" . -}}
<a href="/w/Здравствуйте"><h1>Здравствуйте</h1></a>
{{- template "footer" }}`))

	scrapableTpl := template.Must(template.Must(tpl.Clone()).Parse(`
{{- template "header" . -}}
{{- range .Words -}}
<a href="{{ absWord . }}">{{ .Word }}</a><br/>
{{- end -}}
{{- with .Next }}<a href="{{ . }}">next</a>{{ end -}}
{{- template "footer" -}}`))

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
	arbitAudioCacheDir := filepath.Join(cacheDir, "audio-volatile")
	arbitImgCacheDir := filepath.Join(cacheDir, "img-volatile")
	os.MkdirAll(audioCacheDir, 0700)
	os.MkdirAll(imgCacheDir, 0700)
	os.MkdirAll(arbitAudioCacheDir, 0700)
	os.MkdirAll(arbitImgCacheDir, 0700)

	app := &App{
		prod: Prod,
		// cpurate: make(chan struct{}, 3),
		// netrate: make(chan struct{}, 3),
		conf: Config{
			AudioCacheDir:          audioCacheDir,
			ImageCacheDir:          imgCacheDir,
			ArbitraryAudioCacheDir: arbitAudioCacheDir,
			ArbitraryImageCacheDir: arbitImgCacheDir,
		},
		wordsTpl:     tpl,
		wordTpl:      wordInfoTpl,
		homeTpl:      homeTpl,
		scrapableTpl: scrapableTpl,
		resultsTpl:   resultsTpl,
	}

	// build our own mux since https://github.com/golang/go/issues/21955
	// is a ridiculous issue that should have been fixed in 2017...
	s := tls.New(app.route, l)
	var omux http.Handler
	_, omux = s.OverrideMux(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		omux.ServeHTTP(w, r)
	}))

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
