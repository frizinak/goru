package main

import (
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/tdewolff/minify/v2/js"
)

func open(uri string) (io.ReadCloser, error) {
	if strings.HasPrefix(uri, "http:") || strings.HasPrefix(uri, "https:") {
		res, err := http.Get(uri)
		if err != nil {
			return nil, err
		}

		return res.Body, err
	}

	return os.Open(uri)
}

func main() {
	out := os.Stdout
	for _, a := range os.Args[1:] {
		p := strings.SplitN(a, ":", 2)
		min := false
		if p[0] == "min" {
			min = true
			a = p[1]
		}

		var reader io.Reader
		func() {
			f, err := open(a)
			if err != nil {
				panic(err)
			}
			defer f.Close()
			reader = f
			if min {
				err := js.DefaultMinifier.Minify(nil, out, reader, nil)
				if err != nil {
					panic(err)
				}

				return
			}

			if _, err = io.Copy(out, reader); err != nil {
				panic(err)
			}
		}()
	}
}
