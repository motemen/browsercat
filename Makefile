NPM_BIN = $(shell npm bin)

browsercat: main.go bindata.go
	go build

bindata.go: js/src/main.js node_modules/ansi-sgr-parser/index.js $(NPM_BIN)/browserify
	$(NPM_BIN)/browserify js/src/main.js > js/all.js
	go-bindata -ignore=src js/

node_modules/ansi-sgr-parser/index.js:
	npm install

$(NPM_BIN)/browserify:
	npm install

install: main.go bindata.go
	go install

.PHONY: install
