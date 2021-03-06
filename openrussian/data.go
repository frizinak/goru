package openrussian

import (
	"fmt"
	"strings"
	"sync"
)

type ID uint64

type Stress struct {
	Prefix, Stress, Suffix string
}

func (s Stress) StringMark(mark string) string {
	if s.Stress == "" {
		return s.Prefix
	}
	d := []string{
		s.Prefix,
		s.Stress,
		mark,
		s.Suffix,
	}

	return strings.Join(d, "")
}

func (s Stress) String() string {
	return s.StringMark(string(stressMark))
}

type StressedSentence []Stress

func (s StressedSentence) Join(sep string, mark string) string {
	n := make([]string, len(s))
	for i := range s {
		n[i] = s[i].StringMark(mark)
	}
	return strings.Join(n, sep)
}

func (s StressedSentence) String() string    { return s.Join(" ", sstressMark) }
func (s StressedSentence) StringAlt() string { return s.Join(" ", sstressMarkAlt) }

const (
	stressMark     = '\u0301'
	stressMarkAlt  = '\u0027'
	sstressMark    = string(stressMark)
	sstressMarkAlt = string(stressMarkAlt)
)

type Stressed string
type StressedList []Stressed

func (s Stressed) Unstressed() string {
	n := []rune(s)
	for i := 0; i < len(n); i++ {
		if n[i] == stressMarkAlt || n[i] == stressMark {
			n = append(n[:i], n[i+1:]...)
			i--
		}
	}
	return string(n)
}

func (s Stressed) String() string { return s.Parse().String() }

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

func (s StressedList) String() string {
	l := make(StressedSentence, 0, len(s))
	for _, v := range s {
		l = append(l, v.Parse()...)
	}
	return l.Join(", ", sstressMark)
}

func (s StressedList) Unstressed() string {
	l := make([]string, len(s))
	for i, v := range s {
		l[i] = v.Unstressed()
	}
	return strings.Join(l, ", ")
}

type Aspect uint8

func (g Aspect) String() string { return someAspectsRev[g] }

const (
	AspectBoth Aspect = 1 + iota
	Imperfective
	Perfective
)

var someAspects = map[string]Aspect{
	"both":         AspectBoth,
	"imperfective": Imperfective,
	"perfective":   Perfective,
}

var someAspectsRev = map[Aspect]string{
	AspectBoth:   "both",
	Imperfective: "imperfective",
	Perfective:   "perfective",
}

func aspect(s string) Aspect {
	s = strings.ToLower(s)
	if v, ok := someAspects[s]; ok {
		return v
	}
	return 0
}

type Gender uint8

func (g Gender) String() string { return someGendersRev[g] }

const (
	N Gender = 1 + iota
	F
	M
	Pl
)

var someGenders = map[string]Gender{
	"n":  N,
	"f":  F,
	"m":  M,
	"pl": Pl,
}

var someGendersRev = map[Gender]string{
	N:  "neuter",
	F:  "feminine",
	M:  "masculine",
	Pl: "plural",
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

type Declension struct {
	Nom  StressedList
	Gen  StressedList
	Dat  StressedList
	Acc  StressedList
	Inst StressedList
	Prep StressedList
}

type AdjGenderInfo struct {
	Gender Gender
	Short  StressedList
	Decl   *Declension
}

type AdjInfo struct {
	Comparative StressedList
	Superlative StressedList

	F, M, N, Pl *AdjGenderInfo
}

type NounInfo struct {
	Gender       Gender
	SingularOnly bool
	PluralOnly   bool
}

type Conjugation struct {
	Sg1, Sg2, Sg3, Pl1, Pl2, Pl3 Stressed
}

type VerbInfo struct {
	Aspect Aspect

	ImperativeSg Stressed
	ImperativePl Stressed
	PastM        Stressed
	PastF        Stressed
	PastN        Stressed
	PastPl       Stressed

	Conjugation    *Conjugation
	ActivePresent  *Word
	ActivePast     *Word
	PassivePresent *Word
	PassivePast    *Word
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
	AdjInfo       *AdjInfo
	VerbInfo      *VerbInfo
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

	l              sync.Mutex
	translationMap map[string]int
}

func (t *Translation) initMap() {
	if t.translationMap != nil {
		return
	}
	t.l.Lock()
	if t.translationMap != nil {
		t.l.Unlock()
		return
	}

	t.translationMap = make(map[string]int)
	strs := strings.Split(t.Translation, ",")
	for i, str := range strs {
		k := strings.ToLower(strings.TrimSpace(str))
		if _, ok := t.translationMap[k]; ok {
			continue
		}
		t.translationMap[k] = i
	}
	t.l.Unlock()
}

func (t *Translation) Words() []string {
	t.initMap()
	strs := make([]string, 0, len(t.translationMap))
	for i := range t.translationMap {
		strs = append(strs, i)
	}
	return strs
}

func (t *Translation) HasTranslation(qry string) (bool, int) {
	t.initMap()
	n, ok := t.translationMap[qry]
	return ok, n
}

func (t Translation) String() string {
	return t.Translation
}
