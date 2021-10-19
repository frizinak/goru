package openrussian

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func dec(r io.Reader, row func(int, []string) error) error {
	s := bufio.NewScanner(r)
	s.Split(bufio.ScanLines)
	n := 0
	for s.Scan() {
		n++
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		s := strings.Split(line, "\t")
		if err := row(n, s); err != nil {
			return err
		}
	}
	return s.Err()
}

func parseUint64(d string, optional bool) (uint64, error) {
	if optional && d == "" {
		return 0, nil
	}
	return strconv.ParseUint(d, 10, 64)
}

func parseInt(d string, optional bool) (int, error) {
	if optional && d == "" {
		return 0, nil
	}
	return strconv.Atoi(d)
}

func DecodeWords(r io.Reader) (CSVWords, error) {
	words := make(CSVWords, 10000)
	err := dec(r, func(n int, row []string) error {
		if n == 1 {
			return nil
		}

		if len(row) != 14 {
			r := make([]string, 14)
			copy(r, row)
			row = r
		}

		if row[6] == "1" {
			return nil
		}

		w := CSVWord{}

		id, err := parseUint64(row[0], false)
		if err != nil {
			return err
		}
		pos, err := parseUint64(row[1], true)
		if err != nil {
			return err
		}
		deriv, err := parseUint64(row[4], true)
		if err != nil {
			return err
		}
		rank, err := parseUint64(row[5], true)
		if err != nil {
			return err
		}

		w.ID = ID(id)
		w.Position = pos
		w.Word = row[2]
		w.Stressed = Stressed(row[3])
		if w.Stressed == "" {
			w.Stressed = Stressed(w.Word)
		}
		w.DerivedFrom = ID(deriv)
		w.Rank = rank
		w.Usage = row[8]
		w.WordType = wordType(row[11])
		w.LanguageLevel = languageLevel(row[12])

		if _, ok := words[w.ID]; ok {
			return fmt.Errorf("duplicate word on line %d: id: %d", n, w.ID)
		}
		words[w.ID] = w

		return nil
	})

	return words, err
}

func DecodeTranslations(r io.Reader) (CSVTranslations, error) {
	trans := make(CSVTranslations, 10000)
	err := dec(r, func(n int, row []string) error {
		if n == 1 {
			return nil
		}

		if len(row) != 8 {
			r := make([]string, 8)
			copy(r, row)
			row = r
		}

		if row[1] != "en" {
			return nil
		}

		t := CSVTranslation{}

		id, err := parseUint64(row[0], false)
		if err != nil {
			return err
		}
		word, err := parseUint64(row[2], true)
		if err != nil {
			return err
		}

		t.ID = ID(id)
		t.Word = ID(word)
		t.Translation = row[4]
		t.Example = row[5]
		t.ExampleTranslation = row[6]
		t.Info = row[7]

		if _, ok := trans[t.ID]; ok {
			return fmt.Errorf("duplicate translation on line %d: id: %d", n, t.ID)
		}
		trans[t.ID] = t
		return nil
	})

	return trans, err
}

func DecodeNouns(r io.Reader) (CSVNouns, error) {
	nouns := make(CSVNouns, 10000)
	err := dec(r, func(n int, row []string) error {
		if n == 1 {
			return nil
		}

		if len(row) != 9 {
			r := make([]string, 9)
			copy(r, row)
			row = r
		}

		nn := CSVNoun{}

		id, err := parseUint64(row[0], false)
		if err != nil {
			return err
		}
		declSing, err := parseUint64(row[7], true)
		if err != nil {
			return err
		}
		declPlur, err := parseUint64(row[8], true)
		if err != nil {
			return err
		}

		nn.ID = ID(id)
		nn.Gender = gender(row[1])
		nn.SingularOnly = row[5] == "1"
		nn.PluralOnly = row[6] == "1"
		nn.DeclinationSingular = ID(declSing)
		nn.DeclinationPlural = ID(declPlur)

		if _, ok := nouns[nn.ID]; ok {
			return fmt.Errorf("duplicate noun on line %d: id: %d", n, nn.ID)
		}
		nouns[nn.ID] = nn
		return nil
	})

	return nouns, err
}

func Merge(cw CSVWords, ct CSVTranslations, cn CSVNouns) Words {
	words := make(Words, len(cw))
	for i, w := range cw {
		var noun *NounInfo
		if n, ok := cn[i]; ok {
			noun = &NounInfo{
				Gender:       n.Gender,
				SingularOnly: n.SingularOnly,
				PluralOnly:   n.PluralOnly,
			}
		}

		words[i] = &Word{
			ID:            w.ID,
			Rank:          w.Rank,
			Word:          w.Word,
			Lower:         strings.ToLower(w.Word),
			Stressed:      w.Stressed,
			Translations:  make([]*Translation, 0, 1),
			WordType:      w.WordType,
			LanguageLevel: w.LanguageLevel,
			NounInfo:      noun,
		}
	}

	for i, w := range words {
		w.DerivedFrom = words[cw[i].DerivedFrom]
	}

	for _, t := range ct {
		if _, ok := words[t.Word]; !ok {
			continue
		}

		words[t.Word].Translations = append(
			words[t.Word].Translations,
			&Translation{
				Translation:        t.Translation,
				Example:            t.Example,
				ExampleTranslation: t.ExampleTranslation,
				Info:               t.Info,
			},
		)
	}

	return words
}
