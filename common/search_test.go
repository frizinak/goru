package common

import (
	"sort"
	"testing"

	"github.com/frizinak/goru/dict"
	"github.com/frizinak/goru/openrussian"
)

var d *dict.Dict
var words openrussian.Words
var RuQ = "драствуте"
var EnQ = "thnk you"

func init() {
	var err error
	d, err = GetDict()
	if err != nil {
		panic(err)
	}
	d.InitRussianFuzzIndex()
	d.InitEnglishFuzzIndex()
	words = d.Words()
}

func TestRuQuery(t *testing.T) {
	res := d.SearchRussianFuzzy(RuQ, true, 10)
	if len(res) == 0 || res[0].Word != "здравствуйте" {
		t.Errorf("could not find correct word")
	}
}

func TestEnQuery(t *testing.T) {
	res := d.SearchEnglishFuzzy(EnQ, 10)
	if len(res) == 0 || res[0].Word != "спасибо" {
		t.Errorf("could not find correct word")
	}
}

func BenchmarkRuSearch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		d.Search(RuQ, true, 100)
	}
}

func BenchmarkEnSearch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		d.Search(EnQ, true, 100)
	}
}

func BenchmarkRuSearchFuzzy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		d.SearchFuzzy(RuQ, true, 100)
	}
}

func BenchmarkEnSearchFuzzy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		d.SearchFuzzy(EnQ, true, 100)
	}
}

func BenchmarkRuSearchFuzzyFuzz(b *testing.B) {
	ix := d.GetRussianFuzz()
	for i := 0; i < b.N; i++ {
		ix.Search(RuQ, func(index int, score, low, high uint8) {})
	}
}

func BenchmarkRuSearchFuzzyLevenshtein(b *testing.B) {
	ix := d.GetRussianFuzz()
	lq := uint8(len(RuQ) / 5)
	if lq == 0 {
		lq = 1
	}
	const max = 100

	tmp := make(dict.Results, 0, max)
	ix.Search(RuQ, func(index int, score, low, high uint8) {
		if score >= lq {
			tmp = append(tmp, &dict.Result{Word: &openrussian.Word{}, Score: int(score)})
		}
	})

	for i := 0; i < b.N; i++ {
		results := make(dict.Results, 0, len(tmp))
		for _, w := range tmp {
			w.Levenshtein(RuQ)
			results = append(results, w)
		}
		sort.Sort(results)
	}
}
