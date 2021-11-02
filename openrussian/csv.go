package openrussian

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func SplitStressed(s string) StressedList {
	l := strings.FieldsFunc(s, func(r rune) bool {
		return r == ';' || r == ','
	})
	r := make(StressedList, 0, len(l))
	for _, v := range l {
		v = strings.TrimSpace(v)
		if v != "" {
			r = append(r, Stressed(v))
		}
	}

	return r
}

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

func DecodeAdjectives(r io.Reader) (CSVAdjectives, error) {
	adjs := make(CSVAdjectives, 10000)
	err := dec(r, func(n int, row []string) error {
		if n == 1 {
			return nil
		}

		if len(row) != 12 {
			r := make([]string, 12)
			copy(r, row)
			row = r
		}

		adj := CSVAdjective{}

		id, err := parseUint64(row[0], false)
		if err != nil {
			return err
		}

		var declIDs [4]ID
		ixs := [4]int{8, 9, 10, 11}
		for i, ix := range ixs {
			id, _ := parseUint64(row[ix], true)
			declIDs[i] = ID(id)
		}

		var shorts [4]StressedList
		ixs = [4]int{4, 5, 6, 7}
		for i, ix := range ixs {
			shorts[i] = SplitStressed(row[ix])
		}

		adj.Word = ID(id)

		adj.Incomparable = row[1] == "1"
		adj.Comparative = SplitStressed(row[2])
		adj.Superlative = SplitStressed(row[3])

		adj.DeclM = declIDs[0]
		adj.DeclF = declIDs[1]
		adj.DeclN = declIDs[2]
		adj.DeclPl = declIDs[3]

		adj.ShortM = shorts[0]
		adj.ShortF = shorts[1]
		adj.ShortN = shorts[2]
		adj.ShortPl = shorts[3]

		if _, ok := adjs[adj.Word]; ok {
			return fmt.Errorf("duplicate adjective on line %d: id: %d", n, adj.Word)
		}
		adjs[adj.Word] = adj
		return nil
	})

	return adjs, err
}

func DecodeDeclensions(r io.Reader) (CSVDeclensions, error) {
	decls := make(CSVDeclensions, 10000)
	err := dec(r, func(n int, row []string) error {
		if n == 1 {
			return nil
		}

		if len(row) != 12 {
			r := make([]string, 12)
			copy(r, row)
			row = r
		}

		decl := CSVDeclension{}

		id, err := parseUint64(row[0], false)
		if err != nil {
			return err
		}

		decl.ID = ID(id)

		decl.Nom = SplitStressed(row[2])
		decl.Gen = SplitStressed(row[3])
		decl.Dat = SplitStressed(row[4])
		decl.Acc = SplitStressed(row[5])
		decl.Inst = SplitStressed(row[6])
		decl.Prep = SplitStressed(row[7])

		if _, ok := decls[decl.ID]; ok {
			return fmt.Errorf("duplicate declension on line %d: id: %d", n, decl.ID)
		}
		decls[decl.ID] = decl
		return nil
	})

	return decls, err
}

func DecodeVerbs(r io.Reader) (CSVVerbs, error) {
	verbs := make(CSVVerbs, 10000)
	err := dec(r, func(n int, row []string) error {
		if n == 1 {
			return nil
		}

		if len(row) != 14 {
			r := make([]string, 14)
			copy(r, row)
			row = r
		}

		verb := CSVVerb{}

		id, err := parseUint64(row[0], false)
		if err != nil {
			return err
		}

		var ids [5]ID
		ixs := [5]int{9, 10, 11, 12, 13}
		for i, ix := range ixs {
			id, _ := parseUint64(row[ix], true)
			ids[i] = ID(id)
		}

		verb.Word = ID(id)

		verb.Aspect = aspect(row[1])

		verb.Partner = SplitStressed(row[2])
		verb.ImperativeSg = Stressed(strings.TrimSpace(row[3]))
		verb.ImperativePl = Stressed(strings.TrimSpace(row[4]))
		verb.PastM = Stressed(strings.TrimSpace(row[5]))
		verb.PastF = Stressed(strings.TrimSpace(row[6]))
		verb.PastN = Stressed(strings.TrimSpace(row[7]))
		verb.PastPl = Stressed(strings.TrimSpace(row[8]))

		verb.Conjugation = ids[0]
		verb.ActivePresentWord = ids[1]
		verb.ActivePastWord = ids[2]
		verb.PassivePresentWord = ids[3]
		verb.PassivePastWord = ids[4]

		if _, ok := verbs[verb.Word]; ok {
			return fmt.Errorf("duplicate verbs on line %d: id: %d", n, verb.Word)
		}
		verbs[verb.Word] = verb
		return nil
	})

	return verbs, err
}

func DecodeConjugations(r io.Reader) (CSVConjugations, error) {
	conjs := make(CSVConjugations, 10000)
	err := dec(r, func(n int, row []string) error {
		if n == 1 {
			return nil
		}

		if len(row) != 14 {
			r := make([]string, 14)
			copy(r, row)
			row = r
		}

		conj := CSVConjugation{}

		id, err := parseUint64(row[0], false)
		if err != nil {
			return err
		}

		conj.ID = ID(id)

		conj.Sg1 = Stressed(row[2])
		conj.Sg2 = Stressed(row[3])
		conj.Sg3 = Stressed(row[4])
		conj.Pl1 = Stressed(row[5])
		conj.Pl2 = Stressed(row[6])
		conj.Pl3 = Stressed(row[7])

		if _, ok := conjs[conj.ID]; ok {
			return fmt.Errorf("duplicate conjugation on line %d: id: %d", n, conj.ID)
		}
		conjs[conj.ID] = conj
		return nil
	})

	return conjs, err
}

func Merge(
	cw CSVWords,
	ct CSVTranslations,
	cn CSVNouns,
	ca CSVAdjectives,
	cd CSVDeclensions,
	cv CSVVerbs,
	cc CSVConjugations,
) Words {
	conjs := make(map[ID]*Conjugation, 10000)
	for _, c := range cc {
		conjs[c.ID] = &Conjugation{
			Sg1: c.Sg1, Sg2: c.Sg2, Sg3: c.Sg3,
			Pl1: c.Pl1, Pl2: c.Pl2, Pl3: c.Pl3,
		}
	}

	decls := make(map[ID]*Declension, 10000)
	for _, d := range cd {
		if len(d.Nom)+len(d.Gen)+len(d.Dat)+len(d.Acc)+len(d.Inst)+len(d.Prep) == 0 {
			continue
		}
		decls[d.ID] = &Declension{
			Nom:  d.Nom,
			Gen:  d.Gen,
			Dat:  d.Dat,
			Acc:  d.Acc,
			Inst: d.Inst,
			Prep: d.Prep,
		}
	}

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

		var adj *AdjInfo
		if a, ok := ca[i]; ok {
			adj = &AdjInfo{
				Comparative: a.Comparative,
				Superlative: a.Superlative,
			}

			if len(a.ShortM) != 0 || decls[a.DeclM] != nil {
				adj.M = &AdjGenderInfo{M, a.ShortM, decls[a.DeclM]}
			}
			if len(a.ShortF) != 0 || decls[a.DeclF] != nil {
				adj.F = &AdjGenderInfo{F, a.ShortF, decls[a.DeclF]}
			}
			if len(a.ShortN) != 0 || decls[a.DeclN] != nil {
				adj.N = &AdjGenderInfo{N, a.ShortN, decls[a.DeclN]}
			}
			if len(a.ShortPl) != 0 || decls[a.DeclPl] != nil {
				adj.Pl = &AdjGenderInfo{Pl, a.ShortPl, decls[a.DeclPl]}
			}
		}

		var verb *VerbInfo
		if v, ok := cv[i]; ok {
			verb = &VerbInfo{
				Aspect:       v.Aspect,
				ImperativeSg: v.ImperativeSg,
				ImperativePl: v.ImperativePl,
				PastM:        v.PastM,
				PastF:        v.PastF,
				PastN:        v.PastN,
				PastPl:       v.PastPl,
				Conjugation:  conjs[v.Conjugation],
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
			AdjInfo:       adj,
			VerbInfo:      verb,
		}
	}

	for i, w := range words {
		w.DerivedFrom = words[cw[i].DerivedFrom]
		if w.VerbInfo != nil {
			w.VerbInfo.ActivePresent = words[cv[i].ActivePresentWord]
			w.VerbInfo.ActivePast = words[cv[i].ActivePastWord]
			w.VerbInfo.PassivePresent = words[cv[i].PassivePresentWord]
			w.VerbInfo.PassivePast = words[cv[i].PassivePastWord]
		}
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
