package dict

import (
	"sort"
	"strings"

	"github.com/frizinak/goru/openrussian"
)

const inverseScore = 1<<31 - 1

type Result struct {
	*openrussian.Word
	Match string
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
	if r.Match != "" {
		r.Score = inverseScore - Levenshtein([]rune(r.Match), []rune(qry))
		return
	}
	r.Score = inverseScore - Levenshtein([]rune(r.Word.Word), []rune(qry))
}

func results2words(r []*Result, max int) []*openrussian.Word {
	if max == 0 {
		max = 1000
	}
	if max > len(r) {
		max = len(r)
	}
	r = r[:max]
	w := make([]*openrussian.Word, len(r))
	for i, r := range r {
		w[i] = r.Word
	}
	return w
}

func (d *Dict) Search(qry string, includeWithoutTranslation bool, max int) ([]*openrussian.Word, bool) {
	if IsCyrillic(qry) {
		return d.SearchRussian(qry, includeWithoutTranslation, max), true
	}

	return d.SearchEnglish(qry, max), false
}

func (d *Dict) SearchFuzzy(qry string, includeWithoutTranslation bool, max int) ([]*openrussian.Word, bool) {
	if IsCyrillic(qry) {
		return d.SearchRussianFuzzy(qry, includeWithoutTranslation, max), true
	}

	return d.SearchEnglishFuzzy(qry, max), false
}

func (d *Dict) SearchEnglish(qry string, max int) []*openrussian.Word {
	qry = strings.ToLower(qry)
	results := make(Results, 0)
	for _, w := range d.w {
		if found, ix := w.HasTranslation(qry); found {
			results = append(results, &Result{Word: w, Score: inverseScore - ix})
		}
	}

	sort.Sort(results)
	return results2words(results, max)
}

func (d *Dict) SearchRussian(qry string, includeWithoutTranslation bool, max int) []*openrussian.Word {
	results := make(Results, 0)

	qryLow := strings.ToLower(qry)
	for _, w := range d.w {
		if !includeWithoutTranslation && len(w.Translations) == 0 {
			continue
		}
		if strings.Contains(w.Lower, qryLow) {
			r := &Result{Word: w}
			r.Levenshtein(qry)
			results = append(results, r)
		}
	}

	sort.Sort(results)
	return results2words(results, max)
}

func IsCyrillic(qry string) bool {
	cyrillic := 0
	for _, c := range qry {
		if c >= '\u0400' && c <= '\u04FF' {
			cyrillic++
		}
	}

	return cyrillic >= len(qry)/2
}
