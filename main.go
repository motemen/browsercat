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
<head></head>
<body><pre id="content"></pre></body>
<script>
var content = document.getElementById('content');
var conn = new WebSocket('ws://' + location.host + '/ws');
conn.onmessage = function (e) {
  var message = JSON.parse(e.data);
  if (message.type === 'text') {
    content.innerText += message.data;
  } else {
    console.log(message);
  }
};
</script>
</html>
`

type unit struct{}

type Tee struct {
	Outs map[chan<- string]unit
	cond *sync.Cond
}

func newTee() *Tee {
	return &Tee{
		Outs: map[chan<- string]unit{},
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

func (t *Tee) NewOutChan() <-chan string {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()

	ch := make(chan string)

	t.Outs[ch] = unit{}
	t.cond.Broadcast()

	return ch
}

func (t *Tee) sync() {
	// Wait until there is at least one out chan
	t.cond.L.Lock()
	defer t.cond.L.Unlock()

	if len(t.Outs) == 0 {
		t.cond.Wait()
	}
}

func (t *Tee) Put(data string) {
	t.sync()

	log.Printf("Sending %d bytes to %d chan(s)", len(data), len(t.Outs))
	for ch := range t.Outs {
		ch <- data
	}
}

func (t *Tee) Close() {
	for ch := range t.Outs {
		close(ch)
	}
}

func readLoop(r io.Reader, t *Tee, done chan<- unit) {
	buf := make([]byte, 4096)

	for {
		n, err := r.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("fatal: %s", err)
			}
			break
		}

		t.Put(string(buf[0:n]))
	}

	log.Println("end")

	done <- unit{}
}

type Message struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

func newHTTPServer(t *Tee) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, mainHTML)
	})
	mux.Handle("/ws", websocket.Handler(func(ws *websocket.Conn) {
		ch := t.NewOutChan()
		for bytes := range ch {
			message := Message{Type: "text", Data: bytes}
			websocket.JSON.Send(ws, message)
		}
	}))

	return httptest.NewServer(mux)
}

func main() {
	t := newTee()
	readDone := make(chan unit)

	go readLoop(os.Stdin, t, readDone)

	ts := newHTTPServer(t)
	defer ts.Close()

	fmt.Println(ts.URL)
	openBrowser(ts.URL)

	<-readDone

	log.Printf("done received")

	t.Close()
}

func openBrowser(url string) error {
	return exec.Command("open", url).Run()
}
