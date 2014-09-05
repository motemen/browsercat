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
.foreground-yellow {
  color: yellow;
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
		content, err := Asset("js/all.js")
		if err != nil {
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
