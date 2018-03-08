package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/miekg/dns"
)

type Log struct {
	storage []string
	size    int
	index   int
}

func NewLog(size int) *Log {
	return &Log{
		size:    size,
		storage: make([]string, size),
	}
}

func (l *Log) Store(v string) {
	now := time.Now().UTC().Format("2006/01/02 15:04:05")
	l.storage[l.index%l.size] = fmt.Sprintf("%s %s", now, v)
	l.index++
}

func (l *Log) Reset() {
	for i := 0; i < l.size; i++ {
		l.storage[i] = ""
	}
}

func (l *Log) Do(do func(string)) {
	s := l.index
	for i := 0; i < l.size; i++ {
		do(l.storage[(s+i)%l.size])
	}
}

type server struct {
	l *Log
	m *http.ServeMux
}

func (s server) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	// grab remote server addr for logs
	addr, _, err := net.SplitHostPort(w.RemoteAddr().String())
	if err != nil {
		log.Print(err)
		return
	}

	for _, question := range r.Question {
		v := fmt.Sprintf("%16s   %s", addr, question.String())
		s.l.Store(v)
		log.Print(v)
	}
}

func (s server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain")
	fmt.Fprint(w, "dns log\n")
	s.l.Do(func(x string) {
		if x != "" {
			fmt.Fprintf(w, "%s\n", x)
		}
	})
}

func main() {
	s := server{NewLog(128), http.NewServeMux()}
	go func() {
		log.Fatal(dns.ListenAndServe(":53", "udp", s))
	}()
	s.m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "dns log\n")
		s.l.Do(func(x string) {
			if x != "" {
				fmt.Fprintf(w, "%s\n", x)
			}
		})
	})
	s.m.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		s.l.Reset()
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "reset log\n")
	})

	log.Fatal(http.ListenAndServe(":7000", s.m))
}
