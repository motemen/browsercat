browsercat: main.go bindata.go
	go build

bindata.go: js/src/main.js modules/ansi-sgr-parser/index.js
	browserify js/src/main.js > js/all.js
	go-bindata -ignore=src js/

modules/ansi-sgr-parser/index.js:
	git submodule update --init
