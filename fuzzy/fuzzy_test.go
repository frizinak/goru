package fuzzy

import (
	"testing"
)

func TestLong(t *testing.T) {
	ix := NewIndex(2, []string{
		"short fuzzy word",
		"long fuzzy word aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})

	ix.Search("short fuzy wod", func(index int, score, low, high uint8) {
		if index == 0 && (score != high || score < 5) {
			t.Error("fail 1")
		}
		if index == 1 && (score == high || score != low) {
			t.Error("fail 2")
		}
	})

	ix.Search("aaaaaaaaaaaaaaaaaaaaaaaaaaaa", func(index int, score, low, high uint8) {
		if index == 0 && (score != 0) {
			t.Error("fail 3")
		}
		if index == 1 && (score != high || score < 10) {
			t.Error("fail 4")
		}
	})

	ix.Search(
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		func(index int, score, low, high uint8) {
			if index == 1 && score != 255 {
				t.Error("fail 5")
			}
		},
	)
}
