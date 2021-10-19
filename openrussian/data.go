package openrussian

import (
	"fmt"
	"strings"
)

type ID uint64
type Stressed string

type Stress struct {
	Prefix, Stress, Suffix string
}

func (s Stress) String() string {
	if s.Stress == "" {
		return s.Prefix
	}
	d := []string{
		s.Prefix,
		s.Stress,
		string(stressMark),
		s.Suffix,
	}

	return strings.Join(d, "")
}

type StressedSentence []Stress

func (s StressedSentence) String() string {
	n := make([]string, len(s))
	for i := range s {
		n[i] = s[i].String()
	}
	return strings.Join(n, " ")
}

const (
	stressMark    = '\u0301'
	stressMarkAlt = '\u0027'
)

func (s Stressed) Parse() StressedSentence {
	f := strings.Fields(string(s))
	ss := make(StressedSentence, len(f))
	for i := range f {
		ss[i] = Stressed(f[i]).parse()
	}

	return ss
}

func (s Stressed) parse() Stress {
	found := false
	pref := make([]rune, 0, len(s))
	var stress rune
	suff := make([]rune, 0, len(s))

	for i, c := range s {
		if !found && (c == stressMark || c == stressMarkAlt) {
			if i == 0 {
				continue
			}
			found = true
			stress = pref[len(pref)-1]
			pref = pref[:len(pref)-1]
			continue
		}
		if !found {
			pref = append(pref, c)
			continue
		}
		suff = append(suff, c)
	}

	var stressStr string
	if found {
		stressStr = string(stress)
	}
	return Stress{string(pref), stressStr, string(suff)}
}

type Gender uint8

func (g Gender) String() string { return someGendersRev[g] }

const (
	N Gender = 1 + iota
	F
	M
)

var someGenders = map[string]Gender{
	"n": N,
	"f": F,
	"m": M,
}

var someGendersRev = map[Gender]string{
	N: "neuter",
	F: "feminine",
	M: "masculine",
}

func gender(s string) Gender {
	s = strings.ToLower(s)
	if v, ok := someGenders[s]; ok {
		return v
	}
	return 0
}

type LanguageLevel uint8

func (l LanguageLevel) String() string { return allLanguageLevelsRev[l] }

const (
	A1 LanguageLevel = 1 + iota
	A2
	B1
	B2
	C1
	C2
)

var allLanguageLevels = map[string]LanguageLevel{
	"A1": A1,
	"A2": A2,
	"B1": B1,
	"B2": B2,
	"C1": C1,
	"C2": C2,
}

var allLanguageLevelsRev = map[LanguageLevel]string{
	A1: "A1",
	A2: "A2",
	B1: "B1",
	B2: "B2",
	C1: "C1",
	C2: "C2",
}

func languageLevel(s string) LanguageLevel {
	s = strings.ToUpper(s)
	if v, ok := allLanguageLevels[s]; ok {
		return v
	}
	return 0
}

type WordType uint8

func (w WordType) String() string { return allWordTypesRev[w] }

const (
	Other WordType = iota
	Adjective
	Adverb
	Expression
	Noun
	Verb
)

var allWordTypes = map[string]WordType{
	"other":      Other,
	"adjective":  Adjective,
	"adverb":     Adverb,
	"expression": Expression,
	"noun":       Noun,
	"verb":       Verb,
}

var allWordTypesRev = map[WordType]string{
	Other:      "n/a",
	Adjective:  "adj.",
	Adverb:     "adv.",
	Expression: "expr",
	Noun:       "noun",
	Verb:       "verb",
}

func wordType(s string) WordType {
	s = strings.ToLower(s)
	if v, ok := allWordTypes[s]; ok {
		return v
	}
	return 0
}

type CSVWords map[ID]CSVWord

type CSVWord struct {
	ID            ID
	Position      uint64
	Word          string
	Stressed      Stressed
	DerivedFrom   ID
	Rank          uint64
	Usage         string
	WordType      WordType
	LanguageLevel LanguageLevel
}

type CSVTranslations map[ID]CSVTranslation

type CSVTranslation struct {
	ID                 ID
	Word               ID
	Translation        string
	Example            string
	ExampleTranslation string
	Info               string
}

type CSVNouns map[ID]CSVNoun

type CSVNoun struct {
	ID                  ID
	Gender              Gender
	SingularOnly        bool
	PluralOnly          bool
	DeclinationSingular ID
	DeclinationPlural   ID
}

type NounInfo struct {
	Gender       Gender
	SingularOnly bool
	PluralOnly   bool
}

type Words map[ID]*Word

type Word struct {
	ID            ID
	Rank          uint64
	Word          string
	Lower         string
	Stressed      Stressed
	DerivedFrom   *Word
	Translations  []*Translation
	WordType      WordType
	LanguageLevel LanguageLevel
	NounInfo      *NounInfo
}

func (w *Word) HasTranslation(qry string) (bool, int) {
	smallest := 10000
	found := false
	for _, t := range w.Translations {
		if f, v := t.HasTranslation(qry); f && v < smallest {
			found = true
			smallest = v
		}
	}

	return found, smallest
}

func (w *Word) String() string {
	if w.DerivedFrom == nil {
		return fmt.Sprintf("%s %s", w.Stressed, w.WordType)
	}

	return fmt.Sprintf("%s %s [%s]", w.Stressed, w.WordType, w.DerivedFrom.Stressed)
}

type Translation struct {
	Translation        string
	Example            string
	ExampleTranslation string
	Info               string

	translationMap map[string]int
}

func (t *Translation) HasTranslation(qry string) (bool, int) {
	if t.translationMap == nil {
		t.translationMap = make(map[string]int)
		strs := strings.Split(t.Translation, ",")
		for i, str := range strs {
			k := strings.ToLower(strings.TrimSpace(str))
			if _, ok := t.translationMap[k]; ok {
				continue
			}
			t.translationMap[k] = i
		}
	}
	n, ok := t.translationMap[qry]
	return ok, n
}

func (t Translation) String() string {
	return t.Translation
}
