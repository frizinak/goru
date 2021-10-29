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
	var adj openrussian.CSVAdjectives
	var decl openrussian.CSVDeclensions
	var verbs openrussian.CSVVerbs
	var conjs openrussian.CSVConjugations

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
		{
			f: "temp/adjectives.csv",
			cb: func(r io.Reader) error {
				var err error
				adj, err = openrussian.DecodeAdjectives(r)
				return err
			},
		},
		{
			f: "temp/declensions.csv",
			cb: func(r io.Reader) error {
				var err error
				decl, err = openrussian.DecodeDeclensions(r)
				return err
			},
		},
		{
			f: "temp/verbs.csv",
			cb: func(r io.Reader) error {
				var err error
				verbs, err = openrussian.DecodeVerbs(r)
				return err
			},
		},
		{
			f: "temp/conjugations.csv",
			cb: func(r io.Reader) error {
				var err error
				conjs, err = openrussian.DecodeConjugations(r)
				return err
			},
		},
	}

	gob := "data/data/db.gob"
	gobweb := "data/data/db.web.gob"

	os.MkdirAll("data/data", 0700)
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

	all := openrussian.Merge(words, trans, nouns, nil, nil, nil, nil)
	if err := openrussian.StoreGOB(gob, all); err != nil {
		panic(err)
	}

	all = openrussian.Merge(words, trans, nouns, adj, decl, verbs, conjs)
	if err := openrussian.StoreGOB(gobweb, all); err != nil {
		panic(err)
	}
}
