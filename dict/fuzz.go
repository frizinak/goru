package dict

import (
	"sort"
	"strings"

	"github.com/frizinak/goru/fuzzy"
	"github.com/frizinak/goru/openrussian"
)

func (d *Dict) InitRussianFuzzIndex() {
	if d.rfuzz.index != nil {
		return
	}
	d.rfuzz.l.Lock()
	if d.rfuzz.index != nil {
		d.rfuzz.l.Unlock()
		return
	}

	words := make([]*openrussian.Word, 0, len(d.w))
	l := make([]string, 0, len(d.w))
	for _, w := range d.w {
		words = append(words, w)
		l = append(l, w.Lower)
	}
	d.rfuzz.words = words
	d.rfuzz.index = fuzzy.NewIndex(2, l)
	d.rfuzz.l.Unlock()
}

func (d *Dict) InitEnglishFuzzIndex() {
	if d.efuzz.index != nil {
		return
	}
	d.efuzz.l.Lock()
	if d.efuzz.index != nil {
		d.efuzz.l.Unlock()
		return
	}

	words := make([]*openrussian.Word, 0, len(d.w))
	matches := make([]string, 0, len(d.w))
	l := make([]string, 0, len(d.w))
	for _, w := range d.w {
		for _, t := range w.Translations {
			for _, kw := range t.Words() {
				words = append(words, w)
				matches = append(matches, kw)
				l = append(l, kw)
			}
		}
	}
	d.efuzz.words = words
	d.efuzz.matches = matches
	d.efuzz.index = fuzzy.NewIndex(2, l)
	d.efuzz.l.Unlock()
}

func (d *Dict) GetRussianFuzz() *fuzzy.Index {
	d.InitRussianFuzzIndex()
	return d.rfuzz.index
}

func (d *Dict) GetEnglishFuzz() *fuzzy.Index {
	d.InitEnglishFuzzIndex()
	return d.efuzz.index
}

const levenshteinMax = 500

func (d *Dict) SearchEnglishFuzzy(qry string, max int) []*openrussian.Word {
	d.InitEnglishFuzzIndex()
	if len(qry) > 1<<8-1 {
		qry = qry[:1<<8-1]
	}
	lq := uint8(len(qry) / 3)
	if lq == 0 {
		lq = 1
	}

	tmp := make(Results, 0, max)
	d.efuzz.index.Search(strings.ToLower(qry), func(index int, score, low, high uint8) {
		if score >= lq {
			tmp = append(
				tmp,
				&Result{
					Word:  d.efuzz.words[index],
					Match: d.efuzz.matches[index],
					Score: int(score),
				},
			)
		}
	})

	if len(tmp) > levenshteinMax {
		sort.Sort(tmp)
		tmp = tmp[:levenshteinMax]
	}

	m := make(map[openrussian.ID]*Result, len(tmp))
	for _, w := range tmp {
		w.Levenshtein(qry)
		if ew, ok := m[w.ID]; ok {
			if w.Score > ew.Score {
				m[w.ID] = w
			}
			continue
		}
		m[w.ID] = w
	}

	results := make(Results, 0, len(tmp))
	for _, w := range m {
		results = append(results, w)
	}

	sort.Sort(results)
	return results2words(results, max)
}

func (d *Dict) SearchRussianFuzzy(qry string, includeWithoutTranslation bool, max int) []*openrussian.Word {
	d.InitRussianFuzzIndex()
	if len(qry) > 1<<8-1 {
		qry = qry[:1<<8-1]
	}
	lq := uint8(len(qry) / 5)
	if lq == 0 {
		lq = 1
	}

	tmp := make(Results, 0, max)
	d.rfuzz.index.Search(strings.ToLower(qry), func(index int, score, low, high uint8) {
		if score >= lq {
			tmp = append(tmp, &Result{Word: d.rfuzz.words[index], Score: int(score)})
		}
	})

	if len(tmp) > levenshteinMax {
		sort.Sort(tmp)
		tmp = tmp[:levenshteinMax]
	}

	results := make(Results, 0, len(tmp))
	for _, w := range tmp {
		if !includeWithoutTranslation && len(w.Translations) == 0 {
			continue
		}
		w.Levenshtein(qry)
		results = append(results, w)
	}

	sort.Sort(results)
	return results2words(results, max)
}
