package dict

import "testing"

func TestEdits(t *testing.T) {
	tests := []struct {
		a, b string
		e    string
	}{
		{
			"go russian",
			"hej let's go russion eh?",
			"+h +e +j +  +l +e +t +' +s +  =g =o =  =r =u =s =s =i ~o =n +  +e +h +?",
		},
		{
			"go russian",
			"go russian",
			"=g =o =  =r =u =s =s =i =a =n",
		},
		{
			"go russian",
			"abc go russ",
			"+a +b +c +  =g =o =  =r =u =s =s -i -a -n",
		},
	}

	for _, d := range tests {
		res := LevenshteinEdits([]rune(d.a), []rune(d.b))
		diff := res.DiffString()
		if diff != d.e {
			t.Errorf("edits incorrect for %s - %s\nexp: %s\ngot: %s", d.a, d.b, d.e, diff)
		}
	}
}

var benchS = []rune("здравствуйте")
var benchT = []rune("здраствуйтее")

func BenchmarkLevenshtein(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Levenshtein(benchS, benchT)
	}
}

func BenchmarkLevenshteinEdits(b *testing.B) {
	for i := 0; i < b.N; i++ {
		LevenshteinEdits(benchS, benchT)
	}
}
