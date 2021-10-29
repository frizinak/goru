package openrussian

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

type CSVAdjectives map[ID]CSVAdjective

type CSVAdjective struct {
	Word         ID
	Incomparable bool
	Comparative  StressedList
	Superlative  StressedList

	ShortM  StressedList
	ShortF  StressedList
	ShortN  StressedList
	ShortPl StressedList

	DeclM  ID
	DeclF  ID
	DeclN  ID
	DeclPl ID
}

type CSVDeclensions map[ID]CSVDeclension

type CSVDeclension struct {
	ID   ID
	Nom  StressedList
	Gen  StressedList
	Dat  StressedList
	Acc  StressedList
	Inst StressedList
	Prep StressedList
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

type CSVVerbs map[ID]CSVVerb

type CSVVerb struct {
	Word         ID
	Aspect       Aspect
	Partner      StressedList
	ImperativeSg Stressed
	ImperativePl Stressed
	PastM        Stressed
	PastF        Stressed
	PastN        Stressed
	PastPl       Stressed

	Conjugation        ID
	ActivePresentWord  ID
	ActivePastWord     ID
	PassivePresentWord ID
	PassivePastWord    ID
}

type CSVConjugations map[ID]CSVConjugation

type CSVConjugation struct {
	ID                           ID
	Sg1, Sg2, Sg3, Pl1, Pl2, Pl3 Stressed
}
