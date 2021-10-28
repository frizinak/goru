package dict

import (
	"fmt"
	"sort"
	"strings"

	"github.com/frizinak/goru/fuzzy"
	"github.com/frizinak/goru/openrussian"
)

func levenshteinMatrix(s, t []rune) (func(int, int) int, []int) {
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

	return offset, d
}

func Levenshtein(s, t []rune) int {
	offset, d := levenshteinMatrix(s, t)
	return d[offset(len(s), len(t))]
}

type EditType uint8

const (
	EditNone EditType = iota
	EditAdd
	EditDel
	EditChange
)

type Edit struct {
	Type EditType
	Rune rune
}

func (e Edit) String() string { return string(e.Rune) }

func (e Edit) DiffString() string {
	t := "="
	switch e.Type {
	case EditAdd:
		t = "+"
	case EditDel:
		t = "-"
	case EditChange:
		t = "~"
	}
	return fmt.Sprintf("%s%s", t, string(e.Rune))
}

type Edits []Edit

func (e Edits) String() string {
	l := make([]string, len(e))
	for i := range e {
		l[i] = e[i].String()
	}
	return strings.Join(l, " ")
}

func (e Edits) DiffString() string {
	l := make([]string, len(e))
	for i := range e {
		l[i] = e[i].DiffString()
	}
	return strings.Join(l, " ")
}

func (e Edits) HasEdits() bool {
	for i := range e {
		if e[i].Type != EditNone {
			return true
		}
	}
	return false
}

func LevenshteinEdits(s, t []rune) Edits {
	offset, d := levenshteinMatrix(s, t)
	m := len(s)
	if len(t) > m {
		m = len(t)
	}
	r := make(Edits, 0, m)

	// for j := 0; j <= len(t); j++ {
	// 	for i := 0; i <= len(s); i++ {
	// 		fmt.Printf("%2d ", d[offset(i, j)])
	// 	}
	// 	fmt.Println()
	// }

	var bt func(i, j int)
	bt = func(i, j int) {
		if i == 0 && j == 0 {
			return
		} else if i == 0 && j > 0 {
			r = append(r, Edit{Type: EditAdd, Rune: t[j-1]})
			bt(i, j-1)
			return
		} else if j == 0 && i > 0 {
			r = append(r, Edit{Type: EditDel, Rune: s[i-1]})
			bt(i-1, j)
			return
		} else if s[i-1] == t[j-1] {
			r = append(r, Edit{Type: EditNone, Rune: t[j-1]})
			bt(i-1, j-1)
			return
		}

		n, w, nw := d[offset(i, j-1)], d[offset(i-1, j)], d[offset(i-1, j-1)]
		if n < w && n <= nw {
			r = append(r, Edit{Type: EditAdd, Rune: t[j-1]})
			bt(i, j-1)
			return
		} else if w <= nw {
			r = append(r, Edit{Type: EditDel, Rune: s[i-1]})
			bt(i-1, j)
			return
		}
		r = append(r, Edit{Type: EditChange, Rune: t[j-1]})
		bt(i-1, j-1)
	}

	bt(len(s), len(t))
	for i := 0; i < len(r)/2; i++ {
		j := len(r) - i - 1
		r[i], r[j] = r[j], r[i]
	}

	return r
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

func (d *Dict) Search(qry string, includeWithoutTranslation bool, max int) ([]*openrussian.Word, bool) {
	cyrillic := 0
	for _, c := range qry {
		if c >= '\u0400' && c <= '\u04FF' {
			cyrillic++
		}
	}

	if cyrillic < len(qry)/2 {
		return d.SearchEnglish(qry, max), false
	}

	return d.SearchRussian(qry, includeWithoutTranslation, max), true
}

func (d *Dict) SearchRussianFuzzy(qry string, includeWithoutTranslation bool, max int) []*openrussian.Word {
	if d.fuzz.index == nil {
		words := make([]*openrussian.Word, 0, len(d.w))
		l := make([]string, 0, len(d.w))
		for _, w := range d.w {
			words = append(words, w)
			l = append(l, w.Lower)
		}
		d.fuzz.words = words
		d.fuzz.index = fuzzy.NewIndex(2, l)
	}

	res := d.fuzz.index.Search(strings.ToLower(qry), func(score, low, high float64) bool {
		return score == high
	})

	results := make(Results, 0, len(res))
	for _, ix := range res {
		if !includeWithoutTranslation && len(d.fuzz.words[ix].Translations) == 0 {
			continue
		}
		res := &Result{Word: d.fuzz.words[ix]}
		results = append(results, res)
	}

	sort.Sort(results)
	return results2words(results, max)
}
