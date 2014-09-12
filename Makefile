NPM_BIN = $(shell npm bin)

browsercat: main.go bindata.go
	go build

main.go: deps

bindata.go: assets/main.js assets/main.css
	go-bindata -prefix=assets/ assets/

assets/main.js: js/main.js node_modules assets
	$(NPM_BIN)/browserify js/main.js > assets/main.js

assets/main.css: assets gems
	bundle exec scss style/main.scss > assets/main.css

assets:
	mkdir -p assets

deps:
	go get ./...

node_modules: package.json
	npm install

gems: Gemfile
	bundle install

install: main.go bindata.go
	go install

clean:
	rm -rf browsercat bindata.go assets/

realclean: clean
	rm -rf node_modules

prerequisites:
	which npm >/dev/null 2>&1
	which bundle >/dev/null 2>&1
	which go-bindata >/dev/null 2>&1 || go get github.com/jteeuwen/go-bindata/...

.PHONY: install clean realclean deps prerequisites gems
