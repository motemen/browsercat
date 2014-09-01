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

type chunk []byte

type Tee struct {
	Outs map[chan<- chunk]unit
	cond *sync.Cond
}

func newTee() *Tee {
	return &Tee{
		Outs: map[chan<- chunk]unit{},
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

func (t *Tee) NewOutChan() <-chan chunk {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()

	ch := make(chan chunk)

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

func (t *Tee) Write(p []byte) (int, error) {
	t.sync()

	log.Printf("sending %d bytes to %d chan(s)", len(p), len(t.Outs))

	data := make([]byte, len(p))
	copy(data, p)

	for ch := range t.Outs {
		ch <- chunk(data)
	}

	return len(p), nil
}

func (t *Tee) Close() {
	for ch := range t.Outs {
		close(ch)
	}
}

type Broker struct {
	Reader io.Reader
	Writer io.Writer
	Done   chan unit
}

func newBroker(r io.Reader, w io.Writer) *Broker {
	return &Broker{
		Reader: r,
		Writer: w,
		Done:   make(chan unit),
	}
}

func (l Broker) Do() {
	buf := make([]byte, 4096)

	for {
		n, err := l.Reader.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("fatal: %s", err)
			}
			break
		}

		l.Writer.Write(buf[0:n])
	}

	log.Println("end")

	l.Done <- unit{}
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
	mux.Handle("/ws", websocket.Handler(func(ws *websocket.Conn) {
		ch := tee.NewOutChan()
		for bytes := range ch {
			message := Message{Type: "text", Data: string(bytes)}
			websocket.JSON.Send(ws, message)
		}
	}))

	return httptest.NewServer(mux)
}

func main() {
	tee := newTee()

	broker := newBroker(os.Stdin, tee)
	go broker.Do()

	ts := newHTTPServer(tee)
	defer ts.Close()

	fmt.Println(ts.URL)
	openBrowser(ts.URL)

	<-broker.Done

	log.Printf("done received")

	tee.Close()
}

func openBrowser(url string) error {
	return exec.Command("open", url).Run()
}
