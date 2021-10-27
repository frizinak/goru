TPL := '{{ $$root := .Dir }}{{ range .GoFiles }}{{ printf "%s/%s\n" $$root . }}{{ end }}'

GOB_DEPS = $(shell go list -f '{{ join .Deps "\n" }}' ./cmd/gob)
GOB_FILES = $(shell go list -f $(TPL) ./cmd/gob $(GOB_DEPS))

MIN_DEPS = $(shell go list -f '{{ join .Deps "\n" }}' ./cmd/minify)
MIN_FILES = $(shell go list -f $(TPL) ./cmd/minify $(MIN_DEPS))

EXTRA = data/data/db.gob
EXTRA += data/data/app.js
EXTRA += data/data/*
DEPS = $(shell go list -f '{{ join .Deps "\n" }}' ./cmd/goru)
FILES = $(shell go list -f $(TPL) ./cmd/goru $(DEPS))
FILES += $(EXTRA)

DEPS_WEB = $(shell go list -f '{{ join .Deps "\n" }}' ./cmd/goruweb)
FILES_WEB = $(shell go list -f $(TPL) ./cmd/goruweb $(DEPS_WEB))
FILES_WEB += $(EXTRA)

CSVS = temp/words.csv temp/translations.csv
CSVZIP = temp/openrussian.zip

.PHONY: all
all: dist/goru dist/goruweb

dist/goru: $(FILES)
	go build -o "$@" ./cmd/goru

dist/goruweb: $(FILES_WEB)
	go build -tags web -o "$@" ./cmd/goruweb

.PHONY: install
install: $(FILES)
	go install ./cmd/goru
	go install ./cmd/goruweb

dist/gob: $(GOB_FILES)
	go build -o "$@" ./cmd/gob

data/data/db.gob: dist/gob $(CSVS)
	./dist/gob

dist/minify: $(MIN_FILES)
	go build -o "$@" ./cmd/minify

data/data/app.js: dist/minify src/*.js
	./dist/minify \
		https://cdn.jsdelivr.net/npm/clipboard@2.0.8/dist/clipboard.min.js \
		min:src/app.js > "$@"

$(CSVS):
	@-mkdir temp 2>/dev/null
	[ -f "$(CSVZIP)" ] || curl -Ss https://api.openrussian.org/downloads/openrussian-csv.zip > "$(CSVZIP)"
	unzip -n "$(CSVZIP)" -d temp

.PHONY: clean
clean:
	rm -f data/data/db.gob
	rm -f data/data/app.js
	rm -rf dist

.PHONY: reset
reset: clean
	rm -rf temp


