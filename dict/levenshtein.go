package dict

import (
	"fmt"
	"strings"
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
	r := make(Edits, len(s)+len(t))

	// for j := 0; j <= len(t); j++ {
	// 	for i := 0; i <= len(s); i++ {
	// 		fmt.Printf("%2d ", d[offset(i, j)])
	// 	}
	// 	fmt.Println()
	// }

	ri := len(s) + len(t)
	var bt func(i, j int)
	bt = func(i, j int) {
		ri--
		if i == 0 && j == 0 {
			return
		} else if i == 0 && j > 0 {
			r[ri] = Edit{Type: EditAdd, Rune: t[j-1]}
			bt(i, j-1)
			return
		} else if j == 0 && i > 0 {
			r[ri] = Edit{Type: EditDel, Rune: s[i-1]}
			bt(i-1, j)
			return
		} else if s[i-1] == t[j-1] {
			r[ri] = Edit{Type: EditNone, Rune: t[j-1]}
			bt(i-1, j-1)
			return
		}

		n, w, nw := d[offset(i, j-1)], d[offset(i-1, j)], d[offset(i-1, j-1)]
		if n < w && n <= nw {
			r[ri] = Edit{Type: EditAdd, Rune: t[j-1]}
			bt(i, j-1)
			return
		} else if w <= nw {
			r[ri] = Edit{Type: EditDel, Rune: s[i-1]}
			bt(i-1, j)
			return
		}
		r[ri] = Edit{Type: EditChange, Rune: t[j-1]}
		bt(i-1, j-1)
	}

	bt(len(s), len(t))

	return r[ri+1:]
}
