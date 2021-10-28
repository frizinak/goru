TPL := '{{ $$root := .Dir }}{{ range .GoFiles }}{{ printf "%s/%s\n" $$root . }}{{ end }}'

GOB_DEPS = $(shell go list -f '{{ join .Deps "\n" }}' ./cmd/gob)
GOB_FILES = $(shell go list -f $(TPL) ./cmd/gob $(GOB_DEPS))

MIN_DEPS = $(shell go list -f '{{ join .Deps "\n" }}' ./cmd/minify)
MIN_FILES = $(shell go list -f $(TPL) ./cmd/minify $(MIN_DEPS))

KEYS_DEPS = $(shell go list -f '{{ join .Deps "\n" }}' ./cmd/keys)
KEYS_FILES = $(shell go list -f $(TPL) ./cmd/keys $(KEYS_DEPS))

EXTRA = data/data/db.gob
EXTRA += data/data/app.js
EXTRA += data/data/*

EXTRA_PROD = cmd/goruweb/private.go
EXTRA_PROD += cmd/goruweb/private_account_key
EXTRA_PROD += cmd/goruweb/private_domain_key

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

.PHONY: prod
prod: dist/goruweb-prod

.PHONY: install
install: $(FILES) $(FILE_WEB) $(EXTRA_PROD)
	go install -tags noweb ./cmd/goru
	go install -tags prod ./cmd/goruweb

dist/goruweb-prod: $(FILES_WEB) $(EXTRA_PROD)
	go build -o "$@" -tags prod ./cmd/goruweb

dist/goru: $(FILES)
	go build -tags noweb -o "$@" ./cmd/goru

dist/goruweb: $(FILES_WEB)
	go build -o "$@" ./cmd/goruweb

cmd/goruweb/private.go:
	cp cmd/goruweb/credentials_private.example "$@"

cmd/goruweb/private_account_key: | dist/keys
	./dist/keys rsa "$@"

cmd/goruweb/private_domain_key: | dist/keys
	./dist/keys ecdsa "$@"

dist/keys: $(FILES_KEYS)
	go build -o "$@" ./cmd/keys

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

.PHONY: test
test:
	go test ./...

.PHONY: bench
bench:
	go test -test.benchmem -bench=. ./...
