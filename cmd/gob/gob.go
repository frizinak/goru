package main

import (
	"os"

	"github.com/frizinak/goru/openrussian"
)

func main() {
	gob := "data/db.gob"
	os.Mkdir("data", 0700)
	f, err := os.Open("temp/words.csv")
	if err != nil {
		panic(err)
	}
	w, err := openrussian.DecodeWords(f)
	if err != nil {
		panic(err)
	}
	f.Close()

	f, err = os.Open("temp/translations.csv")
	if err != nil {
		panic(err)
	}
	t, err := openrussian.DecodeTranslations(f)
	if err != nil {
		panic(err)
	}
	f.Close()

	words := openrussian.Merge(w, t)
	if err := openrussian.StoreGOB(gob, words); err != nil {
		panic(err)
	}
}
