NPM_BIN = $(shell npm bin)
SOURCES = main.go tee.go

browsercat: $(SOURCES) bindata.go
	go build

main.go: deps

bindata.go: assets/main.js assets/main.css
	go-bindata -prefix=assets/ assets/

assets/main.js: js/main.js node_modules assets
	$(NPM_BIN)/browserify js/main.js > assets/main.js

assets/main.css: assets node_modules
	$(NPM_BIN)/lessc style/main.less > assets/main.css

assets:
	mkdir -p assets

deps:
	go get ./...

node_modules: package.json
	npm install

install: $(SOURCES) bindata.go
	go install

clean:
	rm -rf browsercat bindata.go assets/

realclean: clean
	rm -rf node_modules

prerequisites:
	which npm >/dev/null 2>&1
	which go-bindata >/dev/null 2>&1 || go get github.com/jteeuwen/go-bindata/...

lint:
	golint $(SOURCES)

.PHONY: install clean realclean deps prerequisites lint
