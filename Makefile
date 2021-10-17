TPL := '{{ $$root := .Dir }}{{ range .GoFiles }}{{ printf "%s/%s\n" $$root . }}{{ end }}'

BINDATA_DEPS = $(shell go list -f '{{ join .Deps "\n" }}' ./cmd/bindata)
BINDATA_FILES = $(shell go list -f $(TPL) ./cmd/bindata $(BINDATA_DEPS))

GOB_DEPS = $(shell go list -f '{{ join .Deps "\n" }}' ./cmd/gob)
GOB_FILES = $(shell go list -f $(TPL) ./cmd/gob $(GOB_DEPS))

DEPS = $(shell go list -f '{{ join .Deps "\n" }}' ./cmd/goru)
FILES = $(shell go list -f $(TPL) ./cmd/goru $(DEPS))

CSVS = temp/words.csv temp/translations.csv
CSVZIP = temp/openrussian.zip

dist/goru: $(FILES) bound/bound.go
	go build -o "$@" ./cmd/goru

.PHONY: install
install: $(FILES) bound/bound.go
	go install ./cmd/goru


dist/bindata: $(BINDATA_FILES)
	go build -o "$@" ./cmd/bindata

dist/gob: $(GOB_FILES)
	go build -o "$@" ./cmd/gob

bound/bound.go: dist/bindata data/db.gob
	./dist/bindata

data/db.gob: dist/gob $(CSVS)
	./dist/gob

$(CSVS):
	@-mkdir temp 2>/dev/null
	[ -f "$(CSVZIP)" ] || curl -Ss https://api.openrussian.org/downloads/openrussian-csv.zip > "$(CSVZIP)"
	unzip -n "$(CSVZIP)" -d temp

.PHONY: clean
clean:
	rm -f data/go.gob
	rm -rf temp
	rm -rf dist

.PHONY: reset
reset: clean
	rm -rf bound


