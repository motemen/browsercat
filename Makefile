NPM_BIN = $(shell npm bin)

browsercat: main.go bindata.go
	go build

bindata.go: assets/main.js npm
	go-bindata -prefix=assets/ assets/

assets/main.js: assets
	$(NPM_BIN)/browserify js/main.js > assets/main.js

assets/main.css: assets
	$(NPM_BIN)/browserify js/main.js > assets/main.js

assets:
	mkdir -p assets

npm:
	npm install

install: main.go bindata.go
	go install

clean:
	rm -f browsercat bindata.go

realclean: clean
	rm -rf node_modules

.PHONY: npm install clean realclean
