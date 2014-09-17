package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"

	"code.google.com/p/go.net/websocket"
	"github.com/docopt/docopt-go"
)

var mainHTML string

func init() {
	styles, err := Asset("main.css")
	if err != nil {
		panic(err)
	}

	mainHTML = `<!DOCTYPE html>
<html>
<style>` + string(styles) + `</style>
<body><pre id="content"></pre></body>
<script src="/js"></script>
</html>`
}

type unit struct{}

type chunk []byte

type Message struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

func newHTTPServer(tee *Tee) *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, mainHTML)
	})

	mux.HandleFunc("/js", func(w http.ResponseWriter, r *http.Request) {
		content, err := Asset("main.js")
		if err != nil {
			log.Printf("server: could not load asset main.js: %s", err)
			w.WriteHeader(500)
		} else {
			w.Write(content)
		}
	})

	mux.Handle("/ws", websocket.Handler(func(ws *websocket.Conn) {
		ch := tee.NewOutChan()

		log.Printf("websocket: new client: %s", ws.RemoteAddr())

		for bytes := range ch {
			message := Message{Type: "text", Data: string(bytes)}
			err := websocket.JSON.Send(ws, message)
			if err != nil {
				log.Printf("websocket: error: %s", err)
				break
			}
		}

		tee.RemoveOutChan(ch)
	}))

	return httptest.NewServer(mux)
}

var version = "0.0"

var usage = `Brings stdout to browsers.

Usage:
  browsercat [--no-open] [--html]

Options:
  --no-open     Do not launch web browser automatically.
  --html        Treat input as raw HTML rather than plain text.
`

func main() {
	arguments, err := docopt.Parse(usage, nil, true, version, false)
	if err != nil {
		panic(err)
	}

	var flagNoOpen, flagHTML bool
	if v, ok := arguments["--no-open"].(bool); ok {
		flagNoOpen = v
	}
	if v, ok := arguments["--html"].(bool); ok {
		flagHTML = v
	}

	tee := newTee()

	server := newHTTPServer(tee)
	defer server.Close()

	url := server.URL
	if flagHTML {
		url = url + "?t=html"
	}

	fmt.Println(url)

	if flagNoOpen {
		// nop
	} else {
		openBrowser(url)
	}

	n, err := io.Copy(tee, os.Stdin)

	log.Printf("main: copying done, sent %d bytes", n)
	if err != nil {
		log.Printf("main: error: %s", err)
	}

	tee.Close()
}

func openBrowser(url string) error {
	return exec.Command("open", url).Run()
}
