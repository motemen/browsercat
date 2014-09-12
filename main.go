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

func main() {
	tee := newTee()

	server := newHTTPServer(tee)
	defer server.Close()

	fmt.Println(server.URL)
	openBrowser(server.URL)

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
