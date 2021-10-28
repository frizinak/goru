package fuzzy

import (
	"strings"
)

type Index struct {
	fuzzyLength int
	n           int
	data        map[string][]int
}

func NewIndex(fuzzyLength int, items []string) *Index {
	if fuzzyLength < 2 {
		fuzzyLength = 2
	}
	ix := &Index{
		fuzzyLength: fuzzyLength,
		n:           len(items),
		data:        make(map[string][]int, len(items)),
	}

	for i, v := range items {
		fuzzy := ix.parts(v)
		for _, p := range fuzzy {
			ix.data[p] = append(ix.data[p], i)
		}
	}

	return ix
}

type Include func(index int, score, low, high uint8)

const maxuint8 = 1<<8 - 1

func (index *Index) Search(q string, include Include) {
	scores := make([]uint8, index.n)
	words := index.parts(q)
	var min, max uint8 = maxuint8, 0
	for _, q := range words {
		if b, ok := index.data[q]; ok {
			for _, ix := range b {
				v := scores[ix]
				if v != maxuint8 {
					v++
				}
				scores[ix] = v
			}
		}
	}

	for _, score := range scores {
		if score < min || min == maxuint8 {
			min = score
		}
		if score > max {
			max = score
		}
	}

	for i, score := range scores {
		include(i, score, min, max)
	}
}

func (index *Index) parts(q string) []string {
	qs := make([]string, 0, len(q))
	p := strings.Fields(
		strings.Trim(strings.TrimSpace(strings.ToLower(q)), "!@#$%^&*=./,"),
	)
	//dupes := make(map[string]struct{})
	add := func(s string) {
		//if _, ok := dupes[s]; !ok {
		qs = append(qs, s)
		//}
	}
	for i := range p {
		v := []rune(p[i])
		if len(v) < 2 {
			continue
		}
		if len(v) <= index.fuzzyLength {
			add(p[i])
			continue
		}
		for j := 0; j < len(v)-index.fuzzyLength+1; j++ {
			add(strings.TrimSpace(string(v[j : j+index.fuzzyLength])))
		}
	}

	return qs
}
