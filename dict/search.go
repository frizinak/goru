package dict

import (
	"sort"
	"strings"

	"github.com/frizinak/goru/openrussian"
)

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

const inverseScore = 1<<31 - 1

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
	r.Score = inverseScore - levenshtein(r.Word.Word, qry)
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

func (d *Dict) SearchEnglish(qry string, max int) []*openrussian.Word {
	qry = strings.ToLower(qry)
	// qryNum, err := strconv.Atoi(qry)
	// qryIsNum := false
	// if err == nil {
	// 	qryIsNum = true
	// }
	results := make(Results, 0)
	for _, w := range d.w {
		// if qryIsNum && w.NumberValue == qryNum {
		// 	results = append(results, &Result{Word: w, Score: inverseScore})
		// }
		if found, ix := w.HasTranslation(qry); found {
			results = append(results, &Result{Word: w, Score: inverseScore - ix})
		}
	}

	sort.Sort(results)
	return results2words(results, max)
}

func (d *Dict) SearchRussian(qry string, includeWithoutTranslation bool, max int) []*openrussian.Word {
	results := make(Results, 0)
	for _, w := range d.w {
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
	return results2words(results, max)
}

func (d *Dict) Search(qry string, includeWithoutTranslation bool, max int) []*openrussian.Word {
	cyrillic := 0
	for _, c := range qry {
		if c >= '\u0400' && c <= '\u04FF' {
			cyrillic++
		}
	}

	if cyrillic < len(qry)/2 {
		return d.SearchEnglish(qry, max)
	}

	return d.SearchRussian(qry, includeWithoutTranslation, max)
}
