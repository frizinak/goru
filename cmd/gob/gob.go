package main

import (
	"io"
	"os"

	"github.com/frizinak/goru/openrussian"
)

func main() {
	var words openrussian.CSVWords
	var trans openrussian.CSVTranslations
	var nouns openrussian.CSVNouns

	x := []struct {
		f  string
		cb func(io.Reader) error
	}{
		{
			f: "temp/words.csv",
			cb: func(r io.Reader) error {
				var err error
				words, err = openrussian.DecodeWords(r)
				return err
			},
		},
		{
			f: "temp/translations.csv",
			cb: func(r io.Reader) error {
				var err error
				trans, err = openrussian.DecodeTranslations(r)
				return err
			},
		},
		{
			f: "temp/nouns.csv",
			cb: func(r io.Reader) error {
				var err error
				nouns, err = openrussian.DecodeNouns(r)
				return err
			},
		},
	}

	gob := "data/db.gob"
	os.Mkdir("data", 0700)
	for _, d := range x {
		func() {
			f, err := os.Open(d.f)
			if err != nil {
				panic(err)
			}
			defer f.Close()
			if err := d.cb(f); err != nil {
				panic(err)
			}
		}()
	}

	all := openrussian.Merge(words, trans, nouns)
	if err := openrussian.StoreGOB(gob, all); err != nil {
		panic(err)
	}
}
