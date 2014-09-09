package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"sync"

	"code.google.com/p/go.net/websocket"
)

var mainHTML = `<!DOCTYPE html>
<html>
<style>
body {
  background-color: rgb(30,30,30);
  color: rgb(247,246,236);
}
.foreground-red {
	color: rgb(207,63,97);
}
.foreground-green {
	color: rgb(123,183,91);
}
.foreground-green .invert {
	background-color: rgb(123,183,91);
	color: rgb(30,30,30);
}
.foreground-green .invert .no-invert {
	color: rgb(123,183,91);
	background-color: rgb(30,30,30);
}
.foreground-yellow {
	color: rgb(233,179,42);
}
.foreground-blue {
	color: rgb(76,154,212);
}
.foreground-magenta {
	color: rgb(165,127,196);
}
.foreground-cyan {
	color: rgb(56,154,173);
}
.foreground-white {
	color: rgb(250,250,246);
}
</style>
<body><pre id="content"></pre></body>
<script src="/js"></script>
</html>
`

type unit struct{}

type chunk []byte

type Tee struct {
	outs map[chan chunk]unit
	cond *sync.Cond
}

func newTee() *Tee {
	return &Tee{
		outs: map[chan chunk]unit{},
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

func (t *Tee) NewOutChan() chan chunk {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()

	ch := make(chan chunk)

	t.outs[ch] = unit{}
	t.cond.Broadcast()

	return ch
}

func (t *Tee) RemoveOutChan(ch chan chunk) {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()

	delete(t.outs, ch)
}

// Wait until there is at least one out chan
func (t *Tee) sync() {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()

	if len(t.outs) == 0 {
		log.Printf("tee: no out chans; waiting for one")
		t.cond.Wait()
	}
}

func (t *Tee) Write(p []byte) (int, error) {
	t.sync()

	log.Printf("tee: sending %d bytes to %d chan(s)", len(p), len(t.outs))

	data := make([]byte, len(p))
	copy(data, p)

	for ch := range t.outs {
		ch <- chunk(data)
	}

	return len(p), nil
}

func (t *Tee) Close() error {
	for ch := range t.outs {
		close(ch)
	}

	return nil
}

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
