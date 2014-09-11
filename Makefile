NPM_BIN = $(shell npm bin)

browsercat: main.go bindata.go
	go build

main.go: deps

bindata.go: assets/main.js assets/main.css
	go-bindata -prefix=assets/ assets/

assets/main.js: assets node_modules
	$(NPM_BIN)/browserify js/main.js > assets/main.js

assets/main.css: assets
	scss style/main.scss > assets/main.css

assets:
	mkdir -p assets

deps:
	go get ./...

node_modules: package.json
	npm install

install: main.go bindata.go
	go install

clean:
	rm -rf browsercat bindata.go assets/

realclean: clean
	rm -rf node_modules

.PHONY: install clean realclean deps
