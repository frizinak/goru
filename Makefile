TPL := '{{ $$root := .Dir }}{{ range .GoFiles }}{{ printf "%s/%s\n" $$root . }}{{ end }}'

GOB_DEPS = $(shell go list -f '{{ join .Deps "\n" }}' ./cmd/gob)
GOB_FILES = $(shell go list -f $(TPL) ./cmd/gob $(GOB_DEPS))

DEPS = $(shell go list -f '{{ join .Deps "\n" }}' ./cmd/goru)
FILES = $(shell go list -f $(TPL) ./cmd/goru $(DEPS))
FILES += data/data/db.gob data/data/LobsterRegular-R7AM.otf data/data/open-sans.regular.ttf

CSVS = temp/words.csv temp/translations.csv
CSVZIP = temp/openrussian.zip

dist/goru: $(FILES)
	go build -o "$@" ./cmd/goru

.PHONY: install
install: $(FILES) bound/bound.go
	go install ./cmd/goru

dist/gob: $(GOB_FILES)
	go build -o "$@" ./cmd/gob

data/data/db.gob: dist/gob $(CSVS)
	./dist/gob

$(CSVS):
	@-mkdir temp 2>/dev/null
	[ -f "$(CSVZIP)" ] || curl -Ss https://api.openrussian.org/downloads/openrussian-csv.zip > "$(CSVZIP)"
	unzip -n "$(CSVZIP)" -d temp

.PHONY: clean
clean:
	rm -f data/data/db.gob
	rm -rf dist

.PHONY: reset
reset: clean
	rm -rf temp


