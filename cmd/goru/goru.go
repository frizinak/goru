package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/frizinak/goru/bound"
	"github.com/frizinak/goru/openrussian"
)

type Result struct {
	*openrussian.Word
	Score int
}

type Results []*Result

func (r Results) Len() int { return len(r) }
func (r Results) Less(i, j int) bool {
	if r[i].Score == r[j].Score {
		if r[i].Rank == r[j].Rank {
			return r[i].Word.Word < r[j].Word.Word
		}

		return r[i].Word.Rank < r[j].Word.Rank
	}

	return r[i].Score > r[j].Score
}

func (r Results) Swap(i, j int) { r[i], r[j] = r[j], r[i] }

func (r *Result) Levenshtein(qry string) {
	r.Score = 10000 - levenshtein(r.Word.Word, qry)
}

func levenshtein(s, t string) int {
	d := make([]int, (len(s)+1)*(len(t)+1))
	stride := len(t) + 1
	offset := func(i, j int) int { return i*stride + j }
	min := func(a, b, c int) int {
		if a < b && a < c {
			return a
		} else if b < c {
			return b
		}

		return c
	}

	for i := 1; i <= len(s); i++ {
		d[offset(i, 0)] = i
	}
	for j := 1; j <= len(t); j++ {
		d[offset(0, j)] = j
	}

	for j := 1; j <= len(t); j++ {
		for i := 1; i <= len(s); i++ {
			cost := 1
			if s[i-1] == t[j-1] {
				cost = 0
			}

			d[offset(i, j)] = min(
				d[offset(i-1, j)]+1,
				d[offset(i, j-1)]+1,
				d[offset(i-1, j-1)]+cost,
			)
		}
	}

	return d[offset(len(s), len(t))]
}

func exit(err error) {
	if err == nil {
		return
	}

	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func searchEnglish(
	qry string,
	max int,
	words openrussian.Words,
) []*openrussian.Word {
	qry = strings.ToLower(qry)
	results := make(Results, 0)
	for _, w := range words {
		if found, ix := w.HasTranslation(qry); found {
			results = append(results, &Result{Word: w, Score: 10000 - ix})
		}
	}
	sort.Sort(results)
	if max == 0 {
		max = 1000
	}
	if max > len(results) {
		max = len(results)
	}
	results = results[:max]
	w := make([]*openrussian.Word, len(results))
	for i, r := range results {
		w[i] = r.Word
	}

	return w
}

func searchRussian(
	qry string,
	includeWithoutTranslation bool,
	max int,
	words openrussian.Words,
) []*openrussian.Word {
	results := make(Results, 0)
	for _, w := range words {
		if !includeWithoutTranslation && len(w.Translations) == 0 {
			continue
		}
		if strings.Contains(w.Word, qry) {
			r := &Result{Word: w}
			r.Levenshtein(qry)
			results = append(results, r)
		}
	}

	sort.Sort(results)
	if max == 0 {
		max = 1000
	}
	if max > len(results) {
		max = len(results)
	}
	results = results[:max]
	w := make([]*openrussian.Word, len(results))
	for i, r := range results {
		w[i] = r.Word
	}

	return w
}

func main() {
	var maxResults uint
	var all bool
	flag.UintVar(&maxResults, "n", 3, "max amount of results")
	flag.BoolVar(&all, "a", false, "include words without translation")
	flag.Parse()

	query := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if query == "" {
		exit(errors.New("please provide a query"))
	}

	a, err := bound.Asset("db.gob")
	if err != nil {
		panic(err)
	}
	r := bytes.NewReader(a)
	words, err := openrussian.DecodeGOB(r)
	if err != nil {
		panic(err)
	}

	cyrillic := 0
	for _, c := range query {
		if c >= '\u0400' && c <= '\u04FF' {
			cyrillic++
		}
	}

	if cyrillic < len(query)/2 {
		results := searchEnglish(query, int(maxResults), words)
		for _, r := range results {
			fmt.Println(r.StringWithTranslations(true, true))
			fmt.Println()
		}

		os.Exit(0)
	}

	results := searchRussian(query, all, int(maxResults), words)
	for _, r := range results {
		fmt.Println(r.StringWithTranslations(true, true))
		fmt.Println()
	}
}
