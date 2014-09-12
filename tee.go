package main

import (
	"log"
	"sync"
)

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
