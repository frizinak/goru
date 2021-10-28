package common

import (
	"sort"
	"testing"

	"github.com/frizinak/goru/dict"
	"github.com/frizinak/goru/openrussian"
)

var d *dict.Dict
var words openrussian.Words
var q = "драствуте"
var longQ string

func init() {
	var err error
	d, err = GetDict()
	if err != nil {
		panic(err)
	}
	d.InitFuzzIndex()
	words = d.Words()

	for i := 0; i < 1000; i++ {
		longQ += q
	}
}

func TestQuery(t *testing.T) {
	res := d.SearchRussianFuzzy(q, true, 10)
	if len(res) == 0 || res[0].Word != "здравствуйте" {
		t.Errorf("could not find correct word")
	}
}

func BenchmarkFullSearch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		d.Search(q, true, 100)
	}
}

func BenchmarkSearchNormal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		d.SearchRussian(q, true, 100)
	}
}

func BenchmarkSearchFuzzy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		d.SearchRussianFuzzy(q, true, 100)
	}
}

func BenchmarkSearchFuzzyFuzz(b *testing.B) {
	ix := d.GetFuzz()
	for i := 0; i < b.N; i++ {
		ix.Search(q, func(index int, score, low, high uint8) {})
	}
}

func BenchmarkSearchFuzzyLevenshtein(b *testing.B) {
	ix := d.GetFuzz()
	lq := uint8(len(q) / 5)
	if lq == 0 {
		lq = 1
	}
	const max = 100

	tmp := make(dict.Results, 0, max)
	ix.Search(q, func(index int, score, low, high uint8) {
		if score >= lq {
			tmp = append(tmp, &dict.Result{Word: &openrussian.Word{}, Score: int(score)})
		}
	})

	for i := 0; i < b.N; i++ {
		results := make(dict.Results, 0, len(tmp))
		for _, w := range tmp {
			w.Levenshtein(q)
			results = append(results, w)
		}
		sort.Sort(results)
	}
}
