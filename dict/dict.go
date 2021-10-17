package dict

import (
	"github.com/frizinak/goru/openrussian"
)

type Dict struct {
	w openrussian.Words
}

func New(w openrussian.Words) *Dict {
	return &Dict{w}
}

func DerivedList(w *openrussian.Word) []*openrussian.Word {
	n := make([]*openrussian.Word, 0, 1)
	derivedList(w, &n)
	return n
}

func derivedList(w *openrussian.Word, list *[]*openrussian.Word) {
	if w.DerivedFrom == nil {
		return
	}

	*list = append(*list, w.DerivedFrom)
	derivedList(w.DerivedFrom, list)
}

func (d *Dict) Words() openrussian.Words { return d.w }
